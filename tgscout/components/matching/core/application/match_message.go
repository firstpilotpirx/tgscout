package application

import (
	"context"
	"errors"
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
	Matches int
}

// MatchMessage is the use case that asks the LLM once per request, saves each
// decision, tells the owner about every match, then marks the message
// processed and moves its chat's cursor.
type MatchMessage struct {
	messages MessageRepository
	llm      Llm
	notifier Notifier
	tracker  *domain.MessageTracker
	delivery Delivery
	health   llmHealth
	locks    keyLocks
}

// NewMatchMessage builds the use case from its ports, the shared tracker and
// the way it retries.
func NewMatchMessage(messages MessageRepository, llm Llm, notifier Notifier, tracker *domain.MessageTracker,
	delivery Delivery) *MatchMessage {
	return &MatchMessage{messages: messages, llm: llm, notifier: notifier, tracker: tracker, delivery: delivery,
		locks: keyLocks{held: map[domain.MessageKey]*keyLock{}}}
}

// Match judges one queued message. On an error the message stays
// unprocessed, the cursor stays and a new message stays in flight.
func (m *MatchMessage) Match(ctx context.Context, item domain.QueueItem) (Outcome, error) {
	key := item.Message.Key
	unlock := m.locks.lock(key)
	defer unlock()
	outcome, err := m.match(ctx, item)
	if !item.Message.Edited {
		m.tracker.Tally().Settle(key, outcome.Matches, err != nil)
	}
	return outcome, err
}

func (m *MatchMessage) match(ctx context.Context, item domain.QueueItem) (Outcome, error) {
	key := item.Message.Key
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
		judgment, err := m.ask(ctx, item, request)
		if err != nil {
			return outcome, err
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
		if err := m.send(ctx, domain.Match{Message: item.Message, Request: request}); err != nil {
			return outcome, err
		}
		if err := m.messages.MarkSent(ctx, key, request.ID, time.Now()); err != nil {
			return outcome, &MatchError{Key: key, Stage: StageStore, Err: err}
		}
		outcome.Matches++
	}
	return outcome, nil
}

// ask judges one request, trying again after a pause until the LLM answers.
// A call the LLM rejects is tried MaxRejectedTries times, then judged «no».
func (m *MatchMessage) ask(ctx context.Context, item domain.QueueItem, request domain.Request) (domain.Judgment, error) {
	key := item.Message.Key
	rejected := 0
	for try := 1; ; try++ {
		judgment, err := m.llm.Judge(ctx, request.Find, item.Message.Text)
		if err == nil {
			if m.health.answered() {
				m.delivery.reporter().LlmBack()
			}
			return judgment, nil
		}
		if ctx.Err() != nil {
			return domain.Judgment{}, &MatchError{Key: key, Stage: StageLlm, Err: err}
		}
		if errors.Is(err, ErrLlmRejected) {
			rejected++
			if rejected >= MaxRejectedTries {
				m.delivery.reporter().GaveUp(item, request, rejected, err)
				return domain.Judgment{}, nil
			}
		} else if m.health.failed() {
			m.delivery.reporter().LlmDown(m.delivery.Backoff.Max)
		}
		if stop := m.delivery.sleep(ctx, m.delivery.Backoff.Pause(try)); stop != nil {
			return domain.Judgment{}, &MatchError{Key: key, Stage: StageLlm, Err: stopped(err, stop)}
		}
	}
}

// send tells the owner about a match, trying again after a pause until it is sent.
func (m *MatchMessage) send(ctx context.Context, match domain.Match) error {
	key := match.Message.Key
	for try := 1; ; try++ {
		err := m.notifier.Notify(ctx, match)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return &MatchError{Key: key, Stage: StageNotify, Err: err}
		}
		pause := m.delivery.Backoff.Pause(try)
		m.delivery.reporter().SendFailed(match, err, pause)
		if stop := m.delivery.sleep(ctx, pause); stop != nil {
			return &MatchError{Key: key, Stage: StageNotify, Err: stopped(err, stop)}
		}
	}
}

// stopped is a failed try whose next try the stop cancelled.
func stopped(err, stop error) error {
	return fmt.Errorf("%w; stopped before the next try: %w", err, stop)
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
