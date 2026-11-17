package application_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

var errBoom = errors.New("boom")

type fakeMessageRepository struct {
	mu        sync.Mutex
	processed map[domain.MessageKey]bool
	sent      map[domain.MessageKey][]string
	decisions []domain.Decision
	cursor    map[int64]int
	cursorAt  map[int64]time.Time
	err       error
}

func newRepository() *fakeMessageRepository {
	return &fakeMessageRepository{
		processed: map[domain.MessageKey]bool{},
		sent:      map[domain.MessageKey][]string{},
		cursor:    map[int64]int{},
		cursorAt:  map[int64]time.Time{},
	}
}

func (r *fakeMessageRepository) IsProcessed(_ context.Context, key domain.MessageKey) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.processed[key], r.err
}

func (r *fakeMessageRepository) SentRequests(_ context.Context, key domain.MessageKey) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.sent[key]...), r.err
}

func (r *fakeMessageRepository) SaveDecision(_ context.Context, decision domain.Decision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decisions = append(r.decisions, decision)
	return nil
}

func (r *fakeMessageRepository) MarkSent(_ context.Context, key domain.MessageKey, requestID string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent[key] = append(r.sent[key], requestID)
	return nil
}

func (r *fakeMessageRepository) MarkProcessed(_ context.Context, key domain.MessageKey, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processed[key] = true
	return nil
}

func (r *fakeMessageRepository) MoveCursor(_ context.Context, chatID int64, messageID int, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if messageID > r.cursor[chatID] {
		r.cursor[chatID] = messageID
	}
	return nil
}

func (r *fakeMessageRepository) Cursor(_ context.Context, chatID int64) (domain.StoredCursor, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, found := r.cursor[chatID]
	return domain.StoredCursor{MessageID: id, At: r.cursorAt[chatID]}, found, r.err
}

type fakeMessageQueue struct{ items []domain.QueueItem }

func (q *fakeMessageQueue) Put(_ context.Context, item domain.QueueItem) error {
	q.items = append(q.items, item)
	return nil
}

type fakeLlm struct {
	mu      sync.Mutex
	answers map[string]domain.Judgment // by request text
	err     error
	calls   []string
}

func (l *fakeLlm) Judge(_ context.Context, find, _ string) (domain.Judgment, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, find)
	return l.answers[find], l.err
}

type fakeNotifier struct {
	matches []domain.Match
	err     error
}

func (n *fakeNotifier) Notify(_ context.Context, match domain.Match) error {
	if n.err != nil {
		return n.err
	}
	n.matches = append(n.matches, match)
	return nil
}
