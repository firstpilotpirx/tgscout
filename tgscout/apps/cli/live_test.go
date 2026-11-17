package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching"
	queue "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// liveSettings is a made-up config: one request on one chat.
type liveSettings struct{ settings domain.Settings }

func (s liveSettings) Settings() domain.Settings { return s.settings }

// memoryMessages keeps decisions in memory and records the processed marks.
type memoryMessages struct {
	mu        sync.Mutex
	processed []domain.MessageKey
	decisions []domain.Decision
}

func (m *memoryMessages) IsProcessed(_ context.Context, key domain.MessageKey) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, each := range m.processed {
		if each == key {
			return true, nil
		}
	}
	return false, nil
}
func (m *memoryMessages) SentRequests(context.Context, domain.MessageKey) ([]string, error) {
	return nil, nil
}
func (m *memoryMessages) SaveDecision(_ context.Context, decision domain.Decision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.decisions = append(m.decisions, decision)
	return nil
}
func (m *memoryMessages) MarkSent(context.Context, domain.MessageKey, string, time.Time) error {
	return nil
}
func (m *memoryMessages) MarkProcessed(_ context.Context, key domain.MessageKey, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed = append(m.processed, key)
	return nil
}
func (m *memoryMessages) MoveCursor(context.Context, int64, int, time.Time) error { return nil }

func (m *memoryMessages) Cursor(context.Context, int64) (domain.StoredCursor, bool, error) {
	return domain.StoredCursor{}, false, nil
}

func (m *memoryMessages) processedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.processed)
}

// yesLlm answers yes to everything.
type yesLlm struct{}

func (yesLlm) Judge(context.Context, string, string) (domain.Judgment, error) {
	return domain.Judgment{Matched: true, Raw: "yes"}, nil
}

// failingListener ends at once with a broken connection.
type failingListener struct{ *fakeTelegram }

func (failingListener) Listen(context.Context, func(context.Context, telegram.Update)) error {
	return errors.New("connection lost")
}

func newTestLive(t *testing.T, con *console, messages *memoryMessages) live {
	t.Helper()
	settings := domain.Settings{Workers: 2, QueueSize: 10, Requests: []domain.Request{
		{ID: "kids-bike", Find: "a kids' bike", Chats: []domain.RequestChat{{Ref: chatRef(t, "@made_up_market")}}},
	}}
	settings.Owner = ownerRef(t)
	items := queue.NewChannelMessageQueue(settings.QueueSize)
	return live{settings: settings, queue: items, workers: settings.Workers, sleep: noSleep,
		ports: matching.Ports{Settings: liveSettings{settings}, Messages: messages, Queue: items, Llm: yesLlm{}}}
}

var marketFound = map[string]telegram.ChatInfo{"@made_up_market": {State: telegram.ChatJoined, ID: 7001,
	Username: "made_up_market", Title: "Made-up market"}}

func noSleep(context.Context, time.Duration) error { return nil }

func ownerRef(t *testing.T) domain.Owner {
	t.Helper()
	owner, err := domain.NewOwner("@made_up_owner")
	require.NoError(t, err)
	return owner
}

func TestServeSaysRunningThenSendsAndPrintsTheMatchesOfTheWatchedChats(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	messages := &memoryMessages{}
	client := &fakeTelegram{updates: []telegram.Update{
		{ChatID: 7001, MessageID: 4211, Date: time.Date(2026, 11, 17, 14, 2, 0, 0, time.Local), Text: "Selling a 16-inch kids' bike, 80 €"},
		{ChatID: 9999, MessageID: 1, Date: time.Now(), Text: "a chat nobody watches"},
	}}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- newTestLive(t, c, messages).serve(ctx, c, client, marketFound) }()
	require.Eventually(t, func() bool { return messages.processedCount() == 1 }, 5*time.Second, 10*time.Millisecond)
	stop()

	require.NoError(t, <-done)
	text := "Match: kids-bike — a kids' bike\nChat: Made-up market\nhttps://t.me/made_up_market/4211\n" +
		"Selling a 16-inch kids' bike, 80 €"
	assert.Equal(t, []string{"@made_up_owner: " + text}, client.sends())
	assert.Equal(t,
		"[run] running — press Ctrl+C to stop\n"+
			"[ok] match sent  14:02  kids-bike\n"+
			"  Match: kids-bike — a kids' bike\n"+
			"  Chat: Made-up market\n"+
			"  https://t.me/made_up_market/4211\n"+
			"  Selling a 16-inch kids' bike, 80 €\n",
		out.String())
	assert.Empty(t, errOut.String())
}

func TestServeReturnsOnlyAfterTheWorkersStopped(t *testing.T) {
	c, _, _ := newTestConsole(true)
	llm := &blockingLlm{entered: make(chan struct{})}
	matcher := newTestLive(t, c, &memoryMessages{})
	matcher.ports.Llm = llm
	client := &fakeTelegram{updates: []telegram.Update{{ChatID: 7001, MessageID: 4211, Date: time.Now(), Text: "a kids' bike"}}}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- matcher.serve(ctx, c, client, marketFound) }()
	<-llm.entered
	stop()

	require.NoError(t, <-done, "the stop is no error")
	assert.True(t, llm.left.Load(), "the worker left the LLM call before serve returned")
}

// blockingLlm holds a judgment until the context ends.
type blockingLlm struct {
	entered chan struct{}
	left    atomic.Bool
}

func (l *blockingLlm) Judge(ctx context.Context, _, _ string) (domain.Judgment, error) {
	close(l.entered)
	<-ctx.Done()
	time.Sleep(20 * time.Millisecond) // a slow exit shows when serve does not wait
	l.left.Store(true)
	return domain.Judgment{}, ctx.Err()
}

func TestServeGivesTheListenerError(t *testing.T) {
	c, _, _ := newTestConsole(true)

	err := newTestLive(t, c, &memoryMessages{}).serve(context.Background(), c, failingListener{&fakeTelegram{}}, marketFound)

	assert.EqualError(t, err, "connection lost")
}

func TestWatchedByIDKeysTheFoundChatsByPeerID(t *testing.T) {
	chats := watchedByID(map[string]telegram.ChatInfo{
		"@made_up_market": {ID: 7001, Username: "made_up_market", Title: "Made-up market"},
		"-1001234567890":  {ID: 1234567890},
	})

	assert.Equal(t, "@made_up_market", chats[7001].Ref.String())
	assert.Equal(t, "made_up_market", chats[7001].Username)
	assert.Equal(t, "Made-up market", chats[7001].Title)
	assert.Equal(t, "-1001234567890", chats[1234567890].Ref.String())
	assert.Empty(t, chats[1234567890].Username)
}
