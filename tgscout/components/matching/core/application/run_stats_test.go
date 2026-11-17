package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// The counts of design D2: what each use case records in the run stats.

var statsStart = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

func cut(stats *domain.RunStats) domain.Status { return stats.Cut(0, statsStart.Add(time.Minute)) }

func TestCollectCountsALiveMessageAndMarksItLive(t *testing.T) {
	f := newCollect(t)
	stats := domain.NewRunStats(statsStart)
	f.useCase.Counting(stats)

	require.NoError(t, f.useCase.Collect(context.Background(), message(t, 1, "Selling a bike")))

	require.Len(t, f.queue.items, 1)
	assert.True(t, f.queue.items[0].Message.Live)
	assert.Equal(t, 1, cut(stats).In)
}

func TestCollectCountsNoEditNoDuplicateAndNoHistoryMessage(t *testing.T) {
	f := newCollect(t)
	stats := domain.NewRunStats(statsStart)
	f.useCase.Counting(stats)
	ctx := context.Background()
	require.NoError(t, f.useCase.Collect(ctx, message(t, 1, "Selling a bike")))
	f.queue.items = nil

	require.NoError(t, f.useCase.Collect(ctx, message(t, 1, "Selling a bike"))) // in flight already
	edit := message(t, 2, "Selling a bike (edited)")
	edit.Edited = true
	require.NoError(t, f.useCase.Collect(ctx, edit))
	_, err := f.useCase.CollectHistory(ctx, message(t, 3, "Selling a lamp"), twoRequests(t).Requests)
	require.NoError(t, err)

	assert.Equal(t, 1, cut(stats).In, "only the first live message counts")
	require.Len(t, f.queue.items, 2)
	for _, each := range f.queue.items {
		assert.False(t, each.Message.Live, "message %d", each.Message.Key.MessageID)
	}
}

func liveItem(t *testing.T, id int, age time.Duration) domain.QueueItem {
	queued := item(t, id)
	queued.Message.Live = true
	queued.Message.SentAt = time.Now().Add(-age)
	return queued
}

func TestMatchRecordsEveryCallTheLatencyAndTheMatch(t *testing.T) {
	f := newMatch()
	stats := domain.NewRunStats(statsStart)
	f.useCase.Counting(stats)

	_, err := f.useCase.Match(context.Background(), f.begin(t, liveItem(t, 1, 3*time.Second)))

	require.NoError(t, err)
	status := cut(stats)
	assert.Equal(t, 1, status.Judged)
	assert.Equal(t, 1, status.Matches)
	assert.True(t, status.HasLlmCall, "two calls recorded")
	require.True(t, status.HasLatency)
	assert.GreaterOrEqual(t, status.Latency, 3*time.Second)
	assert.Less(t, status.Latency, 4*time.Second)
}

func TestMatchJudgesAHistoryMessageWithNoLatency(t *testing.T) {
	f := newMatch()
	stats := domain.NewRunStats(statsStart)
	f.useCase.Counting(stats)

	_, err := f.useCase.Match(context.Background(), f.begin(t, item(t, 1)))

	require.NoError(t, err)
	status := cut(stats)
	assert.Equal(t, 1, status.Judged)
	assert.False(t, status.HasLatency)
}

func TestAFailedJudgmentCountsNoJudgedMessage(t *testing.T) {
	f := newMatch()
	f.repository.err = errors.New("disk full")
	stats := domain.NewRunStats(statsStart)
	f.useCase.Counting(stats)

	_, err := f.useCase.Match(context.Background(), f.begin(t, liveItem(t, 1, time.Second)))

	require.Error(t, err)
	status := cut(stats)
	assert.Equal(t, 0, status.Judged)
	assert.Equal(t, 0, status.Matches)
	assert.False(t, status.HasLatency)
}
