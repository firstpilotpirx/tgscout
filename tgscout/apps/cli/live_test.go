package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching"
	notify "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/notify/console"
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
	items := queue.NewChannelMessageQueue(settings.QueueSize)
	config := matching.NewMatchingConfig(matching.Ports{
		Settings: liveSettings{settings}, Messages: messages, Queue: items,
		Llm: yesLlm{}, Notifier: notify.NewConsoleNotifier(notify.PrintFunc(con.PrintMatch)),
	})
	return live{settings: settings, config: config, queue: items, workers: settings.Workers}
}

var marketFound = map[string]telegram.ChatInfo{"@made_up_market": {State: telegram.ChatJoined, ID: 7001, Username: "made_up_market"}}

func TestServeSaysRunningThenPrintsTheMatchesOfTheWatchedChats(t *testing.T) {
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
	assert.Equal(t,
		"[run] running — press Ctrl+C to stop\n"+
			"[ok] match  14:02  kids-bike  @made_up_market  https://t.me/made_up_market/4211\n"+
			"         Selling a 16-inch kids' bike, 80 €\n",
		out.String())
	assert.Empty(t, errOut.String())
}

func TestServeReturnsOnlyAfterTheWorkersStopped(t *testing.T) {
	c, _, _ := newTestConsole(true)
	ctx, stop := context.WithCancel(context.Background())
	stop()

	err := newTestLive(t, c, &memoryMessages{}).serve(ctx, c, &fakeTelegram{}, marketFound)

	assert.NoError(t, err, "the stop is no error")
}

func TestServeGivesTheListenerError(t *testing.T) {
	c, _, _ := newTestConsole(true)

	err := newTestLive(t, c, &memoryMessages{}).serve(context.Background(), c, failingListener{&fakeTelegram{}}, marketFound)

	assert.EqualError(t, err, "connection lost")
}

func TestWatchedByIDKeysTheFoundChatsByPeerID(t *testing.T) {
	chats := watchedByID(map[string]telegram.ChatInfo{
		"@made_up_market": {ID: 7001, Username: "made_up_market"},
		"-1001234567890":  {ID: 1234567890},
	})

	assert.Equal(t, "@made_up_market", chats[7001].Ref.String())
	assert.Equal(t, "made_up_market", chats[7001].Username)
	assert.Equal(t, "-1001234567890", chats[1234567890].Ref.String())
	assert.Empty(t, chats[1234567890].Username)
}
