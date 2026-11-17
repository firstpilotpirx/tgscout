//go:build integration

package telegram_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

var errRead = errors.New("made-up read error")

// historyClient gives each chat's history and asks to wait once per chat
// named in floods; a chat in broken fails.
type historyClient struct {
	telegram.Client
	mu      sync.Mutex
	history map[int64]telegram.History
	floods  map[int64]time.Duration
	broken  map[int64]bool
	windows []telegram.HistoryWindow
}

func (c *historyClient) ReadHistory(_ context.Context, chatID int64, window telegram.HistoryWindow) (telegram.History, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.windows = append(c.windows, window)
	if wait, ok := c.floods[chatID]; ok {
		delete(c.floods, chatID)
		return telegram.History{}, &telegram.FloodWait{Wait: wait}
	}
	if c.broken[chatID] {
		return telegram.History{}, errRead
	}
	return c.history[chatID], nil
}

// recordingCatchUp records what the collector hands it.
type recordingCatchUp struct {
	opened    []int64
	closed    []int64
	collected []domain.Message
	ranks     []int
	cursor    map[int64]int
	closeErr  error
}

func (r *recordingCatchUp) Open(_ context.Context, chatID int64, ref domain.ChatRef) (domain.CatchUpChat, error) {
	r.opened = append(r.opened, chatID)
	after := r.cursor[chatID]
	return domain.CatchUpChat{ChatID: chatID, Ref: ref, After: after, HasCursor: after > 0,
		Depth: domain.CatchUp{Messages: 500, Days: 3}, Start: time.Now()}, nil
}

func (r *recordingCatchUp) Collect(_ context.Context, _ domain.CatchUpChat, message domain.Message, rank int) error {
	r.collected = append(r.collected, message)
	r.ranks = append(r.ranks, rank)
	return nil
}

func (r *recordingCatchUp) Close(_ context.Context, chat domain.CatchUpChat) error {
	r.closed = append(r.closed, chat.ChatID)
	return r.closeErr
}

func (r *recordingCatchUp) Wait(context.Context) (domain.CatchUpCounts, error) {
	return domain.CatchUpCounts{Read: len(r.collected), Judged: len(r.collected)}, nil
}

func historyChats(t *testing.T) []collect.Chat {
	t.Helper()
	market, err := domain.NewChatRef("@made_up_market")
	require.NoError(t, err)
	street, err := domain.NewChatRef("-1001234567890")
	require.NoError(t, err)
	return []collect.Chat{
		{ID: 42, Ref: market, Username: "made_up_market", Title: "Made-up market"},
		{ID: 1234567890, Ref: street},
	}
}

type recordedReports struct {
	started int
	chats   []collect.ChatReport
	waits   []time.Duration
	summary *collect.Summary
}

func (r *recordedReports) funcs() collect.HistoryReports {
	return collect.HistoryReports{
		OnStart: func(chats int) { r.started = chats },
		OnChat:  func(report collect.ChatReport) { r.chats = append(r.chats, report) },
		OnWait:  func(wait time.Duration) { r.waits = append(r.waits, wait) },
		OnDone:  func(summary collect.Summary) { r.summary = &summary },
	}
}

func noSleep(context.Context, time.Duration) error { return nil }

func TestHistoryCollectorCatchesUpEveryChatInOrder(t *testing.T) {
	useCase := &recordingCatchUp{cursor: map[int64]int{42: 9}}
	reports := &recordedReports{}
	client := &historyClient{history: map[int64]telegram.History{
		42: {Messages: []telegram.Update{
			{ChatID: 42, MessageID: 10, Text: "older"},
			{ChatID: 42, MessageID: 11, Text: "newest"},
		}},
		1234567890: {Messages: []telegram.Update{{ChatID: 1234567890, MessageID: 5, Text: "street"}}, Gap: true},
	}}
	collector := collect.NewHistoryCollector(useCase, reports.funcs(), noSleep)

	require.NoError(t, collector.CatchUp(context.Background(), client, historyChats(t)))

	assert.Equal(t, []int64{42, 1234567890}, useCase.opened)
	assert.Equal(t, []int64{42, 1234567890}, useCase.closed)
	assert.Equal(t, 9, client.windows[0].After)
	assert.Equal(t, 500, client.windows[0].Limit)
	require.Len(t, useCase.collected, 3)
	assert.Equal(t, []int{2, 1, 1}, useCase.ranks)
	assert.Equal(t, "https://t.me/made_up_market/10", useCase.collected[0].Link)
	assert.Equal(t, "Made-up market", useCase.collected[0].ChatTitle)
	assert.Equal(t, "https://t.me/c/1234567890/5", useCase.collected[2].Link)
	assert.Equal(t, 2, reports.started)
	require.Len(t, reports.chats, 2)
	assert.Equal(t, 2, reports.chats[0].Read)
	assert.True(t, reports.chats[0].HasCursor())
	assert.False(t, reports.chats[0].Gap)
	assert.False(t, reports.chats[1].Gap, "a first start has no gap")
	require.NotNil(t, reports.summary)
	assert.Equal(t, 2, reports.summary.Chats)
	assert.Equal(t, 3, reports.summary.Counts.Judged)
}

func TestHistoryCollectorWaitsOutAFloodWait(t *testing.T) {
	useCase := &recordingCatchUp{}
	reports := &recordedReports{}
	client := &historyClient{floods: map[int64]time.Duration{42: time.Second}, history: map[int64]telegram.History{
		42: {Messages: []telegram.Update{{ChatID: 42, MessageID: 1, Text: "after the wait"}}},
	}}
	collector := collect.NewHistoryCollector(useCase, reports.funcs(), noSleep)

	require.NoError(t, collector.CatchUp(context.Background(), client, historyChats(t)[:1]))

	assert.Equal(t, []time.Duration{time.Second}, reports.waits)
	require.Len(t, useCase.collected, 1)
}

func TestHistoryCollectorGoesOnAfterAFailedRead(t *testing.T) {
	useCase := &recordingCatchUp{}
	reports := &recordedReports{}
	client := &historyClient{broken: map[int64]bool{42: true}, history: map[int64]telegram.History{
		1234567890: {Messages: []telegram.Update{{ChatID: 1234567890, MessageID: 5, Text: "street"}}},
	}}
	collector := collect.NewHistoryCollector(useCase, reports.funcs(), noSleep)

	require.NoError(t, collector.CatchUp(context.Background(), client, historyChats(t)))

	assert.Equal(t, []int64{42, 1234567890}, useCase.closed)
	require.Len(t, reports.chats, 2)
	require.ErrorIs(t, reports.chats[0].Err, errRead)
	assert.Len(t, useCase.collected, 1)
}

func TestHistoryCollectorPrintsNoSummaryAfterAStop(t *testing.T) {
	useCase := &recordingCatchUp{}
	reports := &recordedReports{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	collector := collect.NewHistoryCollector(useCase, reports.funcs(), noSleep)

	err := collector.CatchUp(ctx, &historyClient{}, historyChats(t))

	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, reports.summary)
}

func TestHistoryCollectorReportsAFailedClose(t *testing.T) {
	useCase := &recordingCatchUp{closeErr: errRead}
	reports := &recordedReports{}
	collector := collect.NewHistoryCollector(useCase, reports.funcs(), noSleep)

	require.NoError(t, collector.CatchUp(context.Background(), &historyClient{}, historyChats(t)[:1]))

	require.Len(t, reports.chats, 1)
	require.ErrorIs(t, reports.chats[0].Err, errRead, "the stored cursor was not saved")
}
