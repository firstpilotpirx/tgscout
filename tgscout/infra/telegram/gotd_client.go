package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// Config is what the client needs to connect as tgscout's account.
type Config struct {
	APIID   int
	APIHash string
	Storage *FileStorage
}

// gotdClient is the Client over gotd; its logger stays gotd's no-op default.
// Updates go through gotd's updates manager (in-memory state) to a dispatcher.
type gotdClient struct {
	client  *telegram.Client
	gaps    *updates.Manager
	updates *updateSink
	mu      sync.Mutex
	owners  map[string]*tg.InputPeerUser // the found owner's peer, by config text
	peers   map[int64]tg.InputPeerClass  // the found chats' peers, by bare ID
}

func newGotdClient(cfg Config) *gotdClient {
	sink := &updateSink{}
	dispatcher := tg.NewUpdateDispatcher()
	sink.register(dispatcher)
	gaps := updates.New(updates.Config{Handler: dispatcher})
	return &gotdClient{
		client:  telegram.NewClient(cfg.APIID, cfg.APIHash, telegram.Options{SessionStorage: cfg.Storage, UpdateHandler: gaps}),
		gaps:    gaps,
		updates: sink,
		owners:  map[string]*tg.InputPeerUser{},
		peers:   map[int64]tg.InputPeerClass{},
	}
}

func (c *gotdClient) Listen(ctx context.Context, f func(context.Context, Update)) error {
	self, err := c.client.Self(ctx)
	if err != nil {
		return mapError(err)
	}
	c.updates.set(f)
	defer c.updates.set(nil)
	err = c.gaps.Run(ctx, c.client.API(), self.ID, updates.AuthOptions{})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return mapError(err)
}

func (c *gotdClient) Run(ctx context.Context, f func(context.Context) error) error {
	var inner error
	err := c.client.Run(ctx, func(ctx context.Context) error {
		inner = f(ctx)
		return inner
	})
	if inner != nil {
		return inner
	}
	return mapError(err)
}

func (c *gotdClient) Authorized(ctx context.Context) (bool, error) {
	status, err := c.client.Auth().Status(ctx)
	if err != nil {
		return false, mapError(err)
	}
	return status.Authorized, nil
}

func (c *gotdClient) SendCode(ctx context.Context, phone string) (string, error) {
	sent, err := c.client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return "", mapError(err)
	}
	code, ok := sent.(*tg.AuthSentCode)
	if !ok {
		return "", errTelegram
	}
	return code.PhoneCodeHash, nil
}

func (c *gotdClient) SignIn(ctx context.Context, phone, code, hash string) error {
	_, err := c.client.Auth().SignIn(ctx, phone, code, hash)
	return mapError(err)
}

func (c *gotdClient) CheckPassword(ctx context.Context, password string) error {
	_, err := c.client.Auth().Password(ctx, password)
	return mapError(err)
}

func (c *gotdClient) Self(ctx context.Context) (Account, error) {
	user, err := c.client.Self(ctx)
	if err != nil {
		return Account{}, mapError(err)
	}
	return Account{Name: strings.TrimSpace(user.FirstName + " " + user.LastName)}, nil
}

func (c *gotdClient) ResolveChat(ctx context.Context, ref string) (ChatInfo, error) {
	if !strings.HasPrefix(ref, "@") && !strings.Contains(ref, "t.me/") {
		state, title, peer, err := c.dialogState(ctx, ref)
		c.keepPeer(bareChatID(ref), peer)
		return ChatInfo{State: state, ID: bareChatID(ref), Title: title}, err
	}
	resolved, err := c.resolve(ctx, ref)
	if err != nil || resolved == nil {
		return ChatInfo{}, err
	}
	info, peer := chatInfo(resolved)
	c.keepPeer(info.ID, peer)
	return info, nil
}

// keepPeer keeps a found chat's input peer for ReadHistory.
func (c *gotdClient) keepPeer(id int64, peer tg.InputPeerClass) {
	if peer == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peers[id] = peer
}

// chatInfo reads the first chat of a resolved username and its input peer.
func chatInfo(resolved *tg.ContactsResolvedPeer) (ChatInfo, tg.InputPeerClass) {
	for _, chat := range resolved.Chats {
		switch chat := chat.(type) {
		case *tg.Channel:
			return ChatInfo{State: joinedState(chat.Left), ID: chat.ID, Username: chat.Username, Title: chat.Title},
				&tg.InputPeerChannel{ChannelID: chat.ID, AccessHash: chat.AccessHash}
		case *tg.Chat:
			return ChatInfo{State: joinedState(chat.Left), ID: chat.ID, Title: chat.Title},
				&tg.InputPeerChat{ChatID: chat.ID}
		}
	}
	return ChatInfo{}, nil
}

// bareChatID is the peer ID of a numeric reference: "-100…" and "-…" dropped.
func bareChatID(ref string) int64 {
	id, _ := strconv.ParseInt(strings.TrimPrefix(strings.TrimPrefix(ref, "-100"), "-"), 10, 64)
	return id
}

// FindOwner finds the owner's user and keeps its peer for SendToOwner, then
// asks whether the account has a private chat with the owner.
func (c *gotdClient) FindOwner(ctx context.Context, owner string) (OwnerState, error) {
	peer, err := c.ownerPeer(ctx, owner)
	if err != nil || peer == nil {
		return OwnerMissing, err
	}
	c.mu.Lock()
	c.owners[owner] = peer
	c.mu.Unlock()
	open, err := c.chatOpen(ctx, peer)
	if err != nil || !open {
		return OwnerNoChat, err
	}
	return OwnerReady, nil
}

// SendToOwner sends one plain message, link previews off, to the kept owner peer.
func (c *gotdClient) SendToOwner(ctx context.Context, owner, text string) error {
	c.mu.Lock()
	peer := c.owners[owner]
	c.mu.Unlock()
	if peer == nil {
		state, err := c.FindOwner(ctx, owner)
		if err != nil {
			return err
		}
		if state == OwnerMissing {
			return &OwnerNotFoundError{Owner: owner}
		}
		c.mu.Lock()
		peer = c.owners[owner]
		c.mu.Unlock()
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Errorf("random message ID: %w", err)
	}
	_, err := c.client.API().MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer: peer, Message: text, NoWebpage: true, RandomID: int64(binary.LittleEndian.Uint64(random[:])), //nolint:gosec // G115: any 64 bits make a random ID
	})
	return mapError(err)
}

// ownerPeer finds the owner's user: a username through Telegram, a numeric ID
// among the contacts; nil when not found.
func (c *gotdClient) ownerPeer(ctx context.Context, owner string) (*tg.InputPeerUser, error) {
	if strings.HasPrefix(owner, "@") {
		resolved, err := c.resolve(ctx, owner)
		if err != nil || resolved == nil {
			return nil, err
		}
		return firstUserPeer(resolved.Users, 0), nil
	}
	id, err := strconv.ParseInt(owner, 10, 64)
	if err != nil {
		return nil, nil
	}
	contacts, err := c.client.API().ContactsGetContacts(ctx, 0)
	if err != nil {
		return nil, mapError(err)
	}
	list, ok := contacts.(*tg.ContactsContacts)
	if !ok {
		return nil, nil
	}
	return firstUserPeer(list.Users, id), nil
}

// firstUserPeer is the peer of the user with the ID, or of the first user when id is 0.
func firstUserPeer(users []tg.UserClass, id int64) *tg.InputPeerUser {
	for _, user := range users {
		if user, ok := user.(*tg.User); ok && (id == 0 || user.ID == id) {
			return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
		}
	}
	return nil
}

// chatOpen tells whether the account has a private chat with the user that
// holds a message: a user who never wrote has none.
func (c *gotdClient) chatOpen(ctx context.Context, peer *tg.InputPeerUser) (bool, error) {
	dialogs, err := c.client.API().MessagesGetPeerDialogs(ctx, []tg.InputDialogPeerClass{&tg.InputDialogPeer{Peer: peer}})
	if tgerr.Is(err, "PEER_ID_INVALID") {
		return false, nil
	}
	if err != nil {
		return false, mapError(err)
	}
	return hasTopMessage(dialogs.Dialogs), nil
}

func hasTopMessage(dialogs []tg.DialogClass) bool {
	for _, dialog := range dialogs {
		if dialog, ok := dialog.(*tg.Dialog); ok && dialog.TopMessage != 0 {
			return true
		}
	}
	return false
}

// resolve finds a username; a name nobody holds gives nil.
func (c *gotdClient) resolve(ctx context.Context, ref string) (*tg.ContactsResolvedPeer, error) {
	name := ref[strings.LastIndexAny(ref, "@/")+1:]
	resolved, err := c.client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: name})
	if tgerr.Is(err, "USERNAME_NOT_OCCUPIED", "USERNAME_INVALID") {
		return nil, nil
	}
	if err != nil {
		return nil, mapError(err)
	}
	return resolved, nil
}

// dialogState looks a numeric chat ID up among the account's dialogs: a chat
// the account is in is one of them; its title comes from the dialog's entities.
func (c *gotdClient) dialogState(ctx context.Context, ref string) (ChatState, string, tg.InputPeerClass, error) {
	want := strings.TrimPrefix(strings.TrimPrefix(ref, "-100"), "-")
	errFound := errors.New("found")
	title := ""
	var peer tg.InputPeerClass
	err := dialogs.NewQueryBuilder(c.client.API()).GetDialogs().BatchSize(100).ForEach(ctx,
		func(_ context.Context, elem dialogs.Elem) error {
			if peerID(elem.Peer) == want {
				title, peer = dialogTitle(elem), elem.Peer
				return errFound
			}
			return nil
		})
	if errors.Is(err, errFound) {
		return ChatJoined, title, peer, nil
	}
	return ChatMissing, "", nil, mapError(err)
}

func dialogTitle(elem dialogs.Elem) string {
	switch peer := elem.Peer.(type) {
	case *tg.InputPeerChannel:
		if channel, ok := elem.Entities.Channel(peer.ChannelID); ok {
			return channel.Title
		}
	case *tg.InputPeerChat:
		if chat, ok := elem.Entities.Chat(peer.ChatID); ok {
			return chat.Title
		}
	}
	return ""
}

func peerID(peer tg.InputPeerClass) string {
	switch peer := peer.(type) {
	case *tg.InputPeerChannel:
		return strconv.FormatInt(peer.ChannelID, 10)
	case *tg.InputPeerChat:
		return strconv.FormatInt(peer.ChatID, 10)
	}
	return ""
}

func joinedState(left bool) ChatState {
	if left {
		return ChatNotJoined
	}
	return ChatJoined
}
