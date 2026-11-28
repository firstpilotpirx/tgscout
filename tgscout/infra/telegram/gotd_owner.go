package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// FindOwner finds the owner's user and keeps its peer for SendToOwner, then
// asks whether the account has a private chat with the owner.
func (c *gotdClient) FindOwner(ctx context.Context, owner string) (OwnerInfo, error) {
	peer, err := c.ownerPeer(ctx, owner)
	if err != nil || peer == nil {
		return OwnerInfo{}, err
	}
	data := Peer{Kind: PeerUser, ID: peer.UserID, AccessHash: peer.AccessHash}
	state, err := c.keepOwner(ctx, owner, peer)
	return OwnerInfo{State: state, Peer: data}, err
}

// CheckOwner keeps a stored owner's peer for SendToOwner and asks only
// whether the private chat is open; it looks up no name.
func (c *gotdClient) CheckOwner(ctx context.Context, owner string, peer Peer) (OwnerState, error) {
	return c.keepOwner(ctx, owner, &tg.InputPeerUser{UserID: peer.ID, AccessHash: peer.AccessHash})
}

// keepOwner keeps the owner's peer, then asks whether its private chat is open.
func (c *gotdClient) keepOwner(ctx context.Context, owner string, peer *tg.InputPeerUser) (OwnerState, error) {
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
		info, err := c.FindOwner(ctx, owner)
		if err != nil {
			return err
		}
		if info.State == OwnerMissing {
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
