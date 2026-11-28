package telegram

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Errors of the client port; none of them carries text a server sent.
var (
	ErrWrongCode      = errors.New("the login code is wrong")
	ErrWrongPassword  = errors.New("the two-step password is wrong")
	ErrPasswordNeeded = errors.New("the account has a two-step password")
	ErrSessionRevoked = errors.New("the session is no longer accepted by Telegram")
)

// FloodWait means Telegram asks to wait before the same call again.
type FloodWait struct{ Wait time.Duration }

func (e *FloodWait) Error() string { return fmt.Sprintf("Telegram asks to wait %s", e.Wait) }

// Account is the signed-in Telegram account, named by its name only.
type Account struct{ Name string }

// ChatState is what the account knows about one chat of the config.
type ChatState int

// The states of a chat.
const (
	ChatMissing ChatState = iota
	ChatNotJoined
	ChatJoined
)

// ChatInfo is what the account knows about one chat of the config: its
// state, its bare peer ID, its username, empty for a private chat, and its
// title, empty when Telegram gives none. Basic marks a basic group (an
// old-style group, not a supergroup), which Telegram gives no message link.
type ChatInfo struct {
	State    ChatState
	ID       int64
	Username string
	Title    string
	Basic    bool
	// Peer is what a later start needs to read the chat without a lookup.
	Peer Peer
}

// PeerKind is the kind of a Telegram peer.
type PeerKind string

// The kinds of a peer: a channel or supergroup, a basic group, a user.
const (
	PeerChannel    PeerKind = "channel"
	PeerBasicGroup PeerKind = "chat"
	PeerUser       PeerKind = "user"
)

// Peer is a found chat's or user's peer data: its kind, its bare ID and the
// per-account access hash Telegram gave for it (0 for a basic group).
type Peer struct {
	Kind       PeerKind
	ID         int64
	AccessHash int64
}

// OwnerState is what the account knows about the owner.
type OwnerState int

// The states of the owner: unknown to the account, known but with no private
// chat with the account yet, or ready to be written to.
const (
	OwnerMissing OwnerState = iota
	OwnerNoChat
	OwnerReady
)

// OwnerInfo is the found owner's state and peer; the peer is empty when the
// owner is missing.
type OwnerInfo struct {
	State OwnerState
	Peer  Peer
}

// Update is a new or edited message of a chat, a group or a channel.
// ChatID is the bare peer ID; Text is the text or the caption.
type Update struct {
	ChatID    int64
	MessageID int
	Date      time.Time
	Text      string
	Service   bool
	Edited    bool
}

// Client is the port over the MTProto client: the sign-in calls, the
// lookups of the start and the updates. Chats and the owner are the config's
// own text.
type Client interface {
	// Run connects and calls f while the connection lasts.
	Run(ctx context.Context, f func(context.Context) error) error
	Authorized(ctx context.Context) (bool, error)
	// SendCode asks Telegram for a login code and returns its hash.
	SendCode(ctx context.Context, phone string) (string, error)
	SignIn(ctx context.Context, phone, code, hash string) error
	CheckPassword(ctx context.Context, password string) error
	Self(ctx context.Context) (Account, error)
	ResolveChat(ctx context.Context, ref string) (ChatInfo, error)
	// KeepChat keeps a stored chat's peer for ReadHistory; it calls no one.
	KeepChat(info ChatInfo)
	FindOwner(ctx context.Context, owner string) (OwnerInfo, error)
	// CheckOwner keeps a stored owner's peer for SendToOwner and asks only
	// whether the owner's private chat with the account is open.
	CheckOwner(ctx context.Context, owner string, peer Peer) (OwnerState, error)
	// SendToOwner sends one plain text message to the owner; it returns once
	// Telegram confirmed the send. It is the only call that writes.
	SendToOwner(ctx context.Context, owner, text string) error
	// Listen calls f for every new and edited message until ctx ends; it runs
	// inside Run.
	Listen(ctx context.Context, f func(context.Context, Update)) error
	// ReadHistory reads the window of a found chat's history, by its bare
	// peer ID; it runs inside Run and calls Telegram, never writes.
	ReadHistory(ctx context.Context, chatID int64, window HistoryWindow) (History, error)
	// WatchConnection calls f with every drop of the connection and every
	// return after one, on its own goroutine; f should not wait long.
	WatchConnection(f func(ConnectionEvent))
}
