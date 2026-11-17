package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Stages a judgment can fail in.
const (
	StageLlm    = "LLM"
	StageNotify = "send"
	StageStore  = "database"
)

// MatchError is a failed judgment of a message: the message stays unprocessed.
type MatchError struct {
	Key   domain.MessageKey
	Stage string
	Err   error
}

func (e *MatchError) Error() string { return fmt.Sprintf("%s failed: %v", e.Stage, e.Err) }

func (e *MatchError) Unwrap() error { return e.Err }

// Unclear is a judgment the LLM answered neither yes nor no, saved as «no».
type Unclear struct {
	Request domain.Request
	Raw     string
}

// Outcome is what a judged message leaves for the console besides matches.
type Outcome struct {
	Unclear []Unclear
}

// MatchMessage is the use case that asks the LLM once per request, saves each
// decision, tells the owner about every match, then marks the message
// processed and moves its chat's cursor.
type MatchMessage struct {
	messages MessageRepository
	llm      Llm
	notifier Notifier
	tracker  *domain.MessageTracker
	locks    keyLocks
}

// NewMatchMessage builds the use case from its ports and the shared tracker.
func NewMatchMessage(messages MessageRepository, llm Llm, notifier Notifier, tracker *domain.MessageTracker) *MatchMessage {
	return &MatchMessage{messages: messages, llm: llm, notifier: notifier, tracker: tracker,
		locks: keyLocks{held: map[domain.MessageKey]*keyLock{}}}
}

// Match judges one queued message. On an error the message stays
// unprocessed, the cursor stays and a new message stays in flight.
func (m *MatchMessage) Match(ctx context.Context, item domain.QueueItem) (Outcome, error) {
	key := item.Message.Key
	unlock := m.locks.lock(key)
	defer unlock()
	outcome, err := m.judge(ctx, item)
	if err != nil {
		return outcome, m.fail(item, err)
	}
	now := time.Now()
	if err := m.messages.MarkProcessed(ctx, key, now); err != nil {
		return outcome, m.fail(item, &MatchError{Key: key, Stage: StageStore, Err: err})
	}
	if item.Message.Edited {
		return outcome, nil
	}
	if cursor, moved := m.tracker.Finish(key); moved {
		if err := m.messages.MoveCursor(ctx, key.ChatID, cursor, now); err != nil {
			return outcome, &MatchError{Key: key, Stage: StageStore, Err: err}
		}
	}
	return outcome, nil
}

func (m *MatchMessage) fail(item domain.QueueItem, err error) error {
	if !item.Message.Edited {
		m.tracker.Fail(item.Message.Key)
	}
	return err
}

func (m *MatchMessage) judge(ctx context.Context, item domain.QueueItem) (Outcome, error) {
	var outcome Outcome
	key := item.Message.Key
	sent, err := m.messages.SentRequests(ctx, key)
	if err != nil {
		return outcome, &MatchError{Key: key, Stage: StageStore, Err: err}
	}
	for _, request := range withoutSent(item.Requests, sent) {
		judgment, err := m.llm.Judge(ctx, request.Find, item.Message.Text)
		if err != nil {
			return outcome, &MatchError{Key: key, Stage: StageLlm, Err: err}
		}
		if judgment.Unclear {
			outcome.Unclear = append(outcome.Unclear, Unclear{Request: request, Raw: judgment.Raw})
		}
		matched := judgment.Matched && !judgment.Unclear
		decision := domain.Decision{Key: key, RequestID: request.ID, Matched: matched, DecidedAt: time.Now()}
		if err := m.messages.SaveDecision(ctx, decision); err != nil {
			return outcome, &MatchError{Key: key, Stage: StageStore, Err: err}
		}
		if !matched {
			continue
		}
		if err := m.notifier.Notify(ctx, domain.Match{Message: item.Message, Request: request}); err != nil {
			return outcome, &MatchError{Key: key, Stage: StageNotify, Err: err}
		}
		if err := m.messages.MarkSent(ctx, key, request.ID, time.Now()); err != nil {
			return outcome, &MatchError{Key: key, Stage: StageStore, Err: err}
		}
	}
	return outcome, nil
}

// keyLocks holds one lock per message key while some judgment uses it, so an
// edit and its original never interleave.
type keyLocks struct {
	mu   sync.Mutex
	held map[domain.MessageKey]*keyLock
}

type keyLock struct {
	sync.Mutex
	users int
}

func (l *keyLocks) lock(key domain.MessageKey) func() {
	l.mu.Lock()
	lock, ok := l.held[key]
	if !ok {
		lock = &keyLock{}
		l.held[key] = lock
	}
	lock.users++
	l.mu.Unlock()
	lock.Lock()
	return func() {
		lock.Unlock()
		l.mu.Lock()
		lock.users--
		if lock.users == 0 {
			delete(l.held, key)
		}
		l.mu.Unlock()
	}
}
