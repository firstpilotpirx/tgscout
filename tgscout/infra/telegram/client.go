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
	ErrSessionRevoked = errors.New("Telegram no longer accepts the session")
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

// Client is the port over the MTProto client: the sign-in calls and the
// lookups of the start. Chats and the owner are the config's own text.
type Client interface {
	// Run connects and calls f while the connection lasts.
	Run(ctx context.Context, f func(context.Context) error) error
	Authorized(ctx context.Context) (bool, error)
	// SendCode asks Telegram for a login code and returns its hash.
	SendCode(ctx context.Context, phone string) (string, error)
	SignIn(ctx context.Context, phone, code, hash string) error
	CheckPassword(ctx context.Context, password string) error
	Self(ctx context.Context) (Account, error)
	ResolveChat(ctx context.Context, ref string) (ChatState, error)
	FindOwner(ctx context.Context, owner string) (bool, error)
}
