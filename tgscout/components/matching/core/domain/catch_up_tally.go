package domain

import (
	"context"
	"sync"
)

// CatchUpCounts are the numbers of the catch-up summary: messages read,
// skipped (nothing to judge), judged, the matches sent and the messages
// whose judging failed.
type CatchUpCounts struct {
	Read, Skipped, Judged, Matches, Errors int
}

// CatchUpTally counts the history messages of a catch-up and knows which of
// them the judge has not settled yet. A message it does not expect, such as
// a new message, is not counted.
type CatchUpTally struct {
	mu      sync.Mutex
	pending map[MessageKey]bool
	counts  CatchUpCounts
	changed chan struct{}
}

// NewCatchUpTally returns an empty tally.
func NewCatchUpTally() *CatchUpTally {
	return &CatchUpTally{pending: map[MessageKey]bool{}, changed: make(chan struct{})}
}

// Read counts history messages read.
func (t *CatchUpTally) Read(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts.Read += n
}

// Skip counts a history message with nothing to judge.
func (t *CatchUpTally) Skip() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts.Skipped++
}

// Fail counts a history message that failed before the judge got it.
func (t *CatchUpTally) Fail() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts.Errors++
}

// Expect marks a history message handed to the judge.
func (t *CatchUpTally) Expect(key MessageKey) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending[key] = true
}

// Settle ends an expected message: judged with its matches sent, or failed.
func (t *CatchUpTally) Settle(key MessageKey, matches int, failed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.pending[key] {
		return
	}
	delete(t.pending, key)
	if failed {
		t.counts.Errors++
	} else {
		t.counts.Judged++
	}
	t.counts.Matches += matches
	close(t.changed)
	t.changed = make(chan struct{})
}

// Wait returns the counts once every expected message is settled, or the
// context's error when it ends first.
func (t *CatchUpTally) Wait(ctx context.Context) (CatchUpCounts, error) {
	for {
		t.mu.Lock()
		if len(t.pending) == 0 {
			counts := t.counts
			t.mu.Unlock()
			return counts, nil
		}
		changed := t.changed
		t.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return CatchUpCounts{}, ctx.Err()
		}
	}
}
