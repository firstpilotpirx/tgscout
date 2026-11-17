package domain

import "sync"

// MessageTracker is the state the use cases share during a run: the messages
// in flight, the cursor of each chat and the catch-up tally. A failed message stays in flight for
// the rest of the run, so the cursor never passes it and its copies are dropped.
type MessageTracker struct {
	mu       sync.Mutex
	inFlight map[MessageKey]bool
	cursors  map[int64]*Cursor
	tally    *CatchUpTally
}

// NewMessageTracker returns an empty tracker.
func NewMessageTracker() *MessageTracker {
	return &MessageTracker{inFlight: map[MessageKey]bool{}, cursors: map[int64]*Cursor{},
		tally: NewCatchUpTally()}
}

// Tally is the count of the run's catch-up.
func (t *MessageTracker) Tally() *CatchUpTally { return t.tally }

// Begin marks a message in flight; false means it is in flight already.
func (t *MessageTracker) Begin(key MessageKey) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inFlight[key] {
		return false
	}
	t.inFlight[key] = true
	t.cursor(key.ChatID).Begin(key.MessageID)
	return true
}

// Finish ends a message and returns the cursor of its chat and whether it moved.
func (t *MessageTracker) Finish(key MessageKey) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.inFlight, key)
	return t.cursor(key.ChatID).Finish(key.MessageID)
}

// Hold stops the chat's cursor from moving while its history is caught up.
func (t *MessageTracker) Hold(chatID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cursor(chatID).Hold()
}

// Release lets the chat's cursor move again and returns it and whether it moved.
func (t *MessageTracker) Release(chatID int64) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cursor(chatID).Release()
}

// Fail keeps a message in flight for the rest of the run.
func (t *MessageTracker) Fail(key MessageKey) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inFlight[key] = true
}

func (t *MessageTracker) cursor(chatID int64) *Cursor {
	cursor, ok := t.cursors[chatID]
	if !ok {
		cursor = NewCursor()
		t.cursors[chatID] = cursor
	}
	return cursor
}
