package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoryPeers is a made-up store of found chats and owners.
type memoryPeers struct {
	peers   map[PeerRole]map[string]KnownPeer
	deleted []string
	findErr error
}

func newMemoryPeers(peers ...KnownPeer) *memoryPeers {
	store := &memoryPeers{peers: map[PeerRole]map[string]KnownPeer{RoleChat: {}, RoleOwner: {}}}
	for _, peer := range peers {
		store.peers[peer.Role][peer.Ref] = peer
	}
	return store
}

func (s *memoryPeers) FindByRef(_ context.Context, role PeerRole, ref string) (KnownPeer, bool, error) {
	if s.findErr != nil {
		return KnownPeer{}, false, s.findErr
	}
	peer, ok := s.peers[role][ref]
	return peer, ok, nil
}

func (s *memoryPeers) Save(_ context.Context, peer KnownPeer) error {
	s.peers[peer.Role][peer.Ref] = peer
	return nil
}

func (s *memoryPeers) DeleteByRef(_ context.Context, role PeerRole, ref string) error {
	s.deleted = append(s.deleted, string(role)+" "+ref)
	delete(s.peers[role], ref)
	return nil
}

func (s *memoryPeers) has(role PeerRole, ref string) bool {
	_, ok := s.peers[role][ref]
	return ok
}

var (
	storedMarket = KnownPeer{Role: RoleChat, Ref: "@made_up_market", Peer: Peer{Kind: PeerChannel, ID: 7, AccessHash: 70},
		Username: "made_up_market", Title: "Made-up market"}
	storedBasic = KnownPeer{Role: RoleChat, Ref: "-12345", Peer: Peer{Kind: PeerBasicGroup, ID: 12345},
		Title: "Made-up neighbours"}
	storedOwner = KnownPeer{Role: RoleOwner, Ref: "@made_up_owner", Peer: Peer{Kind: PeerUser, ID: 42, AccessHash: 4242},
		Username: "made_up_owner"}
)

func storeLookup(client *stubClient, store *memoryPeers) *Lookup {
	lookup, _, _ := newTestLookup(client)
	lookup.Known = store
	return lookup
}

func TestLookupLooksUpOnlyAChatTheStoreLacks(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_river": ChatJoined},
		ownerChecks: map[string]OwnerState{"@made_up_owner": OwnerReady}}
	store := newMemoryPeers(storedMarket, storedBasic, storedOwner)

	found, counts, err := storeLookup(client, store).Find(context.Background(),
		[]string{"@made_up_market", "-12345", "@made_up_river"}, "@made_up_owner")

	require.NoError(t, err)
	assert.Equal(t, []string{"@made_up_river"}, client.resolved)
	assert.Equal(t, LookupCounts{Stored: 2, LookedUp: 1}, counts)
	assert.Len(t, client.kept, 2)
	assert.Equal(t, ChatInfo{State: ChatJoined, ID: 7, Username: "made_up_market", Title: "Made-up market",
		Peer: storedMarket.Peer}, found["@made_up_market"])
	assert.True(t, found["-12345"].Basic, "a stored kind 'chat' is a basic group")
	assert.False(t, found["@made_up_market"].Basic)
	assert.True(t, store.has(RoleChat, "@made_up_river"))
}

func TestLookupKeepsAStoredReadyOwnerWithoutANameLookup(t *testing.T) {
	client := &stubClient{ownerChecks: map[string]OwnerState{"@made_up_owner": OwnerReady}}
	store := newMemoryPeers(storedMarket, storedOwner)

	_, counts, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market"}, "@made_up_owner")

	require.NoError(t, err)
	assert.Empty(t, client.resolved)
	assert.Empty(t, client.ownersFound)
	assert.Empty(t, store.deleted)
	assert.Equal(t, LookupCounts{Stored: 1}, counts)
}

func TestLookupFindsAStoredOwnerWhoseChatClosedAgain(t *testing.T) {
	client := &stubClient{ownerChecks: map[string]OwnerState{"@made_up_owner": OwnerNoChat},
		owners: map[string]OwnerState{"@made_up_owner": OwnerNoChat}}
	store := newMemoryPeers(storedMarket, storedOwner)

	_, _, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market"}, "@made_up_owner")

	var closed *OwnerChatClosedError
	require.True(t, errors.As(err, &closed), "got %v", err)
	assert.Equal(t, []string{"owner @made_up_owner"}, store.deleted)
	assert.Equal(t, []string{"@made_up_owner"}, client.ownersFound)
}

func TestLookupSavesEveryFoundChatAndTheOwner(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined, "-1001": ChatJoined},
		owners: map[string]OwnerState{"@made_up_owner": OwnerReady}}
	store := newMemoryPeers()

	_, counts, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market", "-1001"},
		"@made_up_owner")

	require.NoError(t, err)
	assert.Equal(t, LookupCounts{LookedUp: 2}, counts)
	assert.Equal(t, KnownPeer{Role: RoleChat, Ref: "@made_up_market", Peer: Peer{Kind: PeerChannel, ID: 7, AccessHash: 70},
		Username: "made_up_market", Title: "Made-up title of made_up_market"}, store.peers[RoleChat]["@made_up_market"])
	assert.True(t, store.has(RoleChat, "-1001"))
	assert.Equal(t, KnownPeer{Role: RoleOwner, Ref: "@made_up_owner", Peer: Peer{Kind: PeerUser, ID: 42, AccessHash: 4242}},
		store.peers[RoleOwner]["@made_up_owner"])
}

func TestLookupNeverSavesAMissingOrNotJoinedChat(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined, "@made_up_river": ChatNotJoined},
		owners: map[string]OwnerState{"@made_up_owner": OwnerReady}}
	store := newMemoryPeers()

	_, _, err := storeLookup(client, store).Find(context.Background(),
		[]string{"@made_up_market", "@made_up_river", "@no_such_chat"}, "@made_up_owner")

	var notFound *ChatsNotFoundError
	require.True(t, errors.As(err, &notFound))
	assert.True(t, store.has(RoleChat, "@made_up_market"))
	assert.False(t, store.has(RoleChat, "@made_up_river"))
	assert.False(t, store.has(RoleChat, "@no_such_chat"))
}

func TestLookupSavesAChatFoundAfterAWait(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined},
		owners: map[string]OwnerState{"@made_up_owner": OwnerReady},
		floods: map[string]time.Duration{"@made_up_market": 12 * time.Second}}
	store := newMemoryPeers()

	_, counts, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market"}, "@made_up_owner")

	require.NoError(t, err)
	assert.True(t, store.has(RoleChat, "@made_up_market"))
	assert.Equal(t, LookupCounts{LookedUp: 1}, counts)
}

func TestLookupNeverSavesAMissingOwner(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined}}
	store := newMemoryPeers()

	_, _, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market"}, "123456789")

	var owner *OwnerNotFoundError
	require.True(t, errors.As(err, &owner))
	assert.False(t, store.has(RoleOwner, "123456789"))
}

func TestLookupSavesAFoundOwnerWhoNeverWrote(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined},
		owners: map[string]OwnerState{"@made_up_owner": OwnerNoChat}}
	store := newMemoryPeers()

	_, _, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market"}, "@made_up_owner")

	var closed *OwnerChatClosedError
	require.True(t, errors.As(err, &closed))
	assert.True(t, store.has(RoleOwner, "@made_up_owner"), "a found owner is saved (design D4)")
}

func TestLookupStopsOnAStoreError(t *testing.T) {
	broken := errors.New("made-up disk error")
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined},
		owners: map[string]OwnerState{"@made_up_owner": OwnerReady}}
	store := newMemoryPeers()
	store.findErr = broken

	_, _, err := storeLookup(client, store).Find(context.Background(), []string{"@made_up_market"}, "@made_up_owner")

	assert.ErrorIs(t, err, broken)
}
