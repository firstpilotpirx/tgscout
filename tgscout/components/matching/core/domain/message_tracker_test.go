package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func key(chat int64, id int) domain.MessageKey { return domain.MessageKey{ChatID: chat, MessageID: id} }

func TestTrackerBeginsAKeyOnce(t *testing.T) {
	tracker := domain.NewMessageTracker()

	assert.True(t, tracker.Begin(key(1, 10)))
	assert.False(t, tracker.Begin(key(1, 10)))
}

func TestTrackerCursorNeverPassesAnUnfinishedMessage(t *testing.T) {
	tracker := domain.NewMessageTracker()
	for _, id := range []int{10, 11, 12} {
		tracker.Begin(key(1, id))
	}

	cursor, moved := tracker.Finish(key(1, 10))
	assert.True(t, moved)
	assert.Equal(t, 10, cursor)

	_, moved = tracker.Finish(key(1, 12))
	assert.False(t, moved)

	cursor, moved = tracker.Finish(key(1, 11))
	assert.True(t, moved)
	assert.Equal(t, 12, cursor)
}

func TestTrackerFailedKeyBlocksTheCursorAndItsCopies(t *testing.T) {
	tracker := domain.NewMessageTracker()
	tracker.Begin(key(1, 10))
	tracker.Begin(key(1, 11))

	tracker.Fail(key(1, 10))
	_, moved := tracker.Finish(key(1, 11))

	assert.False(t, moved)
	assert.False(t, tracker.Begin(key(1, 10)))
}

func TestTrackerKeepsTheChatsApart(t *testing.T) {
	tracker := domain.NewMessageTracker()
	tracker.Begin(key(1, 10))
	tracker.Begin(key(2, 5))

	cursor, moved := tracker.Finish(key(2, 5))

	assert.True(t, moved)
	assert.Equal(t, 5, cursor)
	assert.True(t, tracker.Begin(key(2, 10)))
}

func TestCursorMovesPastFinishedMessagesOnly(t *testing.T) {
	cursor := domain.NewCursor()
	cursor.Begin(3)
	cursor.Begin(4)

	_, moved := cursor.Finish(4)
	assert.False(t, moved)
	position, moved := cursor.Finish(3)
	assert.True(t, moved)
	assert.Equal(t, 4, position)
}
