package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MissingChat is a chat of the config the account cannot watch, and why.
type MissingChat struct{ Chat, Reason string }

// ChatsNotFoundError lists every chat the account cannot watch.
type ChatsNotFoundError struct{ Chats []MissingChat }

func (e *ChatsNotFoundError) Error() string {
	names := make([]string, 0, len(e.Chats))
	for _, chat := range e.Chats {
		names = append(names, chat.Chat)
	}
	return fmt.Sprintf("%d chats not found: %s", len(e.Chats), strings.Join(names, ", "))
}

// OwnerNotFoundError means Telegram does not know the owner for the account.
type OwnerNotFoundError struct{ Owner string }

func (e *OwnerNotFoundError) Error() string { return "owner not found: " + e.Owner }

// Lookup finds the watched chats and the owner, waiting whenever Telegram
// asks to.
type Lookup struct {
	Client  Client
	Waiting func(time.Duration)
	Sleep   func(context.Context, time.Duration) error
}

// Find looks up every chat and the owner and returns the found chats by
// their config text; it returns every problem at once.
func (l *Lookup) Find(ctx context.Context, chats []string, owner string) (map[string]ChatInfo, error) {
	var missing []MissingChat
	found := map[string]ChatInfo{}
	for _, chat := range chats {
		info, err := retry(ctx, l, func() (ChatInfo, error) { return l.Client.ResolveChat(ctx, chat) })
		if err != nil {
			return nil, err
		}
		switch info.State {
		case ChatMissing:
			missing = append(missing, MissingChat{Chat: chat, Reason: "no such chat"})
		case ChatNotJoined:
			missing = append(missing, MissingChat{Chat: chat, Reason: "the account has not joined it"})
		case ChatJoined:
			found[chat] = info
		}
	}
	var chatsErr, ownerErr error
	if len(missing) > 0 {
		chatsErr = &ChatsNotFoundError{Chats: missing}
	}
	ownerFound, err := retry(ctx, l, func() (bool, error) { return l.Client.FindOwner(ctx, owner) })
	if err != nil {
		return nil, err
	}
	if !ownerFound {
		ownerErr = &OwnerNotFoundError{Owner: owner}
	}
	if err := errors.Join(chatsErr, ownerErr); err != nil {
		return nil, err
	}
	return found, nil
}

// retry repeats call after every FLOOD_WAIT, reporting and sleeping the wait.
func retry[T any](ctx context.Context, l *Lookup, call func() (T, error)) (T, error) {
	for {
		value, err := call()
		var flood *FloodWait
		if !errors.As(err, &flood) {
			return value, err
		}
		l.Waiting(flood.Wait)
		if err := l.Sleep(ctx, flood.Wait); err != nil {
			return value, err
		}
	}
}

// SleepContext waits d or until ctx ends.
func SleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
