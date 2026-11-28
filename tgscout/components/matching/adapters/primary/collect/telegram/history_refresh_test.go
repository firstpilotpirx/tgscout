package telegram_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// madeUpMarketID is the bare peer ID of the made-up stored chat.
const madeUpMarketID = 42

// memoryKnownPeers is a made-up store of found chats and owners.
type memoryKnownPeers struct {
	mu    sync.Mutex
	peers map[string]telegram.KnownPeer
}

func (s *memoryKnownPeers) key(role telegram.PeerRole, ref string) string {
	return string(role) + " " + ref
}

func (s *memoryKnownPeers) FindByRef(_ context.Context, role telegram.PeerRole, ref string) (telegram.KnownPeer, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	peer, ok := s.peers[s.key(role, ref)]
	return peer, ok, nil
}

func (s *memoryKnownPeers) Save(_ context.Context, peer telegram.KnownPeer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers[s.key(peer.Role, peer.Ref)] = peer
	return nil
}

func (s *memoryKnownPeers) DeleteByRef(_ context.Context, role telegram.PeerRole, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, s.key(role, ref))
	return nil
}

// goneClient answers the first gone history reads with ErrChatGone, then the
// history; a lookup by name gives resolved.
type goneClient struct {
	telegram.Client
	mu       sync.Mutex
	gone     int
	reads    int
	history  telegram.History
	resolved telegram.ChatInfo
	lookups  []string
}

func (c *goneClient) ReadHistory(context.Context, int64, telegram.HistoryWindow) (telegram.History, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	if c.gone > 0 {
		c.gone--
		return telegram.History{}, telegram.ErrChatGone
	}
	return c.history, nil
}

func (c *goneClient) ResolveChat(_ context.Context, ref string) (telegram.ChatInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lookups = append(c.lookups, ref)
	return c.resolved, nil
}

func (c *goneClient) KeepChat(telegram.ChatInfo) {}

// storedMarket is a store that holds the made-up market chat.
func storedMarket() *memoryKnownPeers {
	store := &memoryKnownPeers{peers: map[string]telegram.KnownPeer{}}
	store.peers[store.key(telegram.RoleChat, "@made_up_market")] = telegram.KnownPeer{
		Role: telegram.RoleChat, Ref: "@made_up_market", Username: "made_up_market", Title: "Made-up market",
		Peer: telegram.Peer{Kind: telegram.PeerChannel, ID: madeUpMarketID, AccessHash: 7},
	}
	return store
}

// refreshRun reads the stored market once by catch-up with a lookup that
// can look it up again.
type refreshRun struct {
	useCase   *recordingRefreshCatchUp
	reports   []collect.ChatReport
	refreshed []collect.Chat
}

func catchUpWithRefresh(t *testing.T, client *goneClient, store *memoryKnownPeers) refreshRun {
	t.Helper()
	market, err := domain.NewChatRef("@made_up_market")
	require.NoError(t, err)
	run := refreshRun{useCase: &recordingRefreshCatchUp{}}
	lookup := &telegram.Lookup{Client: client, Known: store, Waiting: func(time.Duration) {},
		Sleep: func(context.Context, time.Duration) error { return nil }}
	collector := collect.NewHistoryCollector(run.useCase, collect.HistoryReports{
		OnChat:      func(report collect.ChatReport) { run.reports = append(run.reports, report) },
		OnRefreshed: func(chat collect.Chat, _ bool) { run.refreshed = append(run.refreshed, chat) },
	}, func(context.Context, time.Duration) error { return nil }).WithRefresher(lookup)
	chats := []collect.Chat{{ID: madeUpMarketID, Ref: market, Username: "made_up_market", Title: "Made-up market"}}

	require.NoError(t, collector.CatchUp(context.Background(), client, chats))
	return run
}

func TestCatchUpFailsAStoredChatTheAccountLeft(t *testing.T) {
	client := &goneClient{gone: 2, resolved: telegram.ChatInfo{State: telegram.ChatNotJoined, ID: madeUpMarketID,
		Username: "made_up_market"}}
	store := storedMarket()

	run := catchUpWithRefresh(t, client, store)

	require.Len(t, run.reports, 1)
	require.ErrorIs(t, run.reports[0].Err, telegram.ErrChatGone)
	assert.Empty(t, run.useCase.closed, "a failed read keeps the cursor where it was")
	assert.Equal(t, []string{"@made_up_market"}, client.lookups, "looked up by name once")
	_, held, err := store.FindByRef(context.Background(), telegram.RoleChat, "@made_up_market")
	require.NoError(t, err)
	assert.False(t, held, "the store no longer holds the chat, so the next start looks it up by name")
}

func TestCatchUpLooksUpAStoredChatTelegramNoLongerKnowsAndReadsItAgain(t *testing.T) {
	client := &goneClient{gone: 1,
		resolved: telegram.ChatInfo{State: telegram.ChatJoined, ID: madeUpMarketID, Username: "made_up_market",
			Title: "Made-up market", Peer: telegram.Peer{Kind: telegram.PeerChannel, ID: madeUpMarketID, AccessHash: 8}},
		history: telegram.History{Messages: []telegram.Update{{ChatID: madeUpMarketID, MessageID: 3, Text: "a kids bike"}}}}
	store := storedMarket()

	run := catchUpWithRefresh(t, client, store)

	require.Len(t, run.refreshed, 1, "one line that the chat was looked up again")
	assert.Equal(t, int64(madeUpMarketID), run.refreshed[0].ID)
	require.Len(t, run.reports, 1)
	require.NoError(t, run.reports[0].Err)
	assert.Equal(t, 1, run.reports[0].Read)
	require.Len(t, run.useCase.collected, 1)
	assert.Equal(t, []int64{madeUpMarketID}, run.useCase.closed)
	assert.Equal(t, 2, client.reads)
	stored, held, err := store.FindByRef(context.Background(), telegram.RoleChat, "@made_up_market")
	require.NoError(t, err)
	require.True(t, held, "the chat found again is stored again")
	assert.Equal(t, int64(8), stored.Peer.AccessHash)
}

func TestCatchUpFailsAStoredChatFoundAgainUnderAnotherID(t *testing.T) {
	client := &goneClient{gone: 1, resolved: telegram.ChatInfo{State: telegram.ChatJoined, ID: 43,
		Username: "made_up_market", Peer: telegram.Peer{Kind: telegram.PeerChannel, ID: 43, AccessHash: 9}}}
	store := storedMarket()

	run := catchUpWithRefresh(t, client, store)

	require.Len(t, run.reports, 1)
	require.ErrorIs(t, run.reports[0].Err, telegram.ErrChatGone)
	assert.Empty(t, run.refreshed)
	_, held, err := store.FindByRef(context.Background(), telegram.RoleChat, "@made_up_market")
	require.NoError(t, err)
	assert.False(t, held)
}

// recordingRefreshCatchUp records what the collector opens, hands on and closes.
type recordingRefreshCatchUp struct {
	mu        sync.Mutex
	collected []domain.Message
	closed    []int64
}

func (r *recordingRefreshCatchUp) Open(_ context.Context, chatID int64, ref domain.ChatRef) (domain.CatchUpChat, error) {
	return domain.CatchUpChat{ChatID: chatID, Ref: ref, Depth: domain.CatchUp{Messages: 500, Days: 3},
		Start: time.Now()}, nil
}

func (r *recordingRefreshCatchUp) Collect(_ context.Context, _ domain.CatchUpChat, message domain.Message, _ int) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collected = append(r.collected, message)
	return true, nil
}

func (r *recordingRefreshCatchUp) Close(_ context.Context, chat domain.CatchUpChat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = append(r.closed, chat.ChatID)
	return nil
}

func (r *recordingRefreshCatchUp) Pause([]int64) {}

func (r *recordingRefreshCatchUp) Wait(context.Context) (domain.CatchUpCounts, error) {
	return domain.CatchUpCounts{}, nil
}
