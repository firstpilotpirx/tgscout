package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var historyNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

// newestFirst gives messages from id down to 1, one per hour back from now.
func newestFirst(id int) []Update {
	var updates []Update
	for each := id; each >= 1; each-- {
		updates = append(updates, Update{ChatID: 7, MessageID: each, Date: historyNow.Add(-time.Duration(id-each) * time.Hour)})
	}
	return updates
}

func messageIDs(history History) []int {
	var ids []int
	for _, each := range history.Messages {
		ids = append(ids, each.MessageID)
	}
	return ids
}

func TestCutWindowReadsAfterTheCursorOldestFirst(t *testing.T) {
	history := cutWindow(newestFirst(10), HistoryWindow{After: 7, Limit: 500, Since: historyNow.Add(-72 * time.Hour)})

	assert.Equal(t, []int{8, 9, 10}, messageIDs(history))
	assert.False(t, history.Gap)
}

func TestCutWindowStopsAtTheLimit(t *testing.T) {
	history := cutWindow(newestFirst(10), HistoryWindow{Limit: 3, Since: historyNow.Add(-72 * time.Hour)})

	assert.Equal(t, []int{8, 9, 10}, messageIDs(history))
	assert.True(t, history.Gap)
}

func TestCutWindowStopsAtTheDays(t *testing.T) {
	history := cutWindow(newestFirst(10), HistoryWindow{After: 2, Limit: 500, Since: historyNow.Add(-90 * time.Minute)})

	assert.Equal(t, []int{9, 10}, messageIDs(history))
	assert.True(t, history.Gap)
}

func TestCutWindowWithTheLimitReachingTheCursorHasNoGap(t *testing.T) {
	history := cutWindow(newestFirst(10), HistoryWindow{After: 7, Limit: 3, Since: historyNow.Add(-72 * time.Hour)})

	assert.Equal(t, []int{8, 9, 10}, messageIDs(history))
	assert.False(t, history.Gap)
}

func TestPageHistoryFetchesUntilTheWindowIsFull(t *testing.T) {
	all := newestFirst(250)
	var offsets []int
	fetch := func(_ context.Context, offsetID int) ([]Update, error) {
		offsets = append(offsets, offsetID)
		start := 0
		if offsetID > 0 {
			start = 250 - offsetID + 1
		}
		return all[start:min(start+100, len(all))], nil
	}

	history, err := pageHistory(context.Background(), fetch, HistoryWindow{Limit: 120, Since: historyNow.Add(-1000 * time.Hour)})

	require.NoError(t, err)
	assert.Equal(t, []int{0, 151}, offsets)
	assert.Len(t, history.Messages, 120)
	assert.Equal(t, 131, history.Messages[0].MessageID)
	assert.True(t, history.Gap)
}

func TestPageHistoryStopsOnAnEmptyPage(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, offsetID int) ([]Update, error) {
		calls++
		if offsetID == 0 {
			return newestFirst(10)[:5], nil // 10 down to 6; older ones were deleted
		}
		return nil, nil
	}

	history, err := pageHistory(context.Background(), fetch, HistoryWindow{Limit: 500, Since: historyNow.Add(-1000 * time.Hour)})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Equal(t, []int{6, 7, 8, 9, 10}, messageIDs(history))
	assert.False(t, history.Gap)
}

func TestPageHistoryReturnsAFetchError(t *testing.T) {
	boom := errors.New("boom")
	fetch := func(context.Context, int) ([]Update, error) { return nil, boom }

	_, err := pageHistory(context.Background(), fetch, HistoryWindow{Limit: 5})

	assert.ErrorIs(t, err, boom)
}
