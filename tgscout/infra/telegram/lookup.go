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

// OwnerChatClosedError means the owner never wrote to the account, so the
// account cannot write to the owner.
type OwnerChatClosedError struct{ Owner string }

func (e *OwnerChatClosedError) Error() string { return "cannot write to the owner: " + e.Owner }

// PeerRole tells what a stored peer is to tgscout.
type PeerRole string

// The roles of a stored peer: a watched chat or the owner.
const (
	RoleChat  PeerRole = "chat"
	RoleOwner PeerRole = "owner"
)

// KnownPeer is a chat or the owner found at an earlier start, by its config text.
type KnownPeer struct {
	Role     PeerRole
	Ref      string
	Peer     Peer
	Username string
	Title    string
}

// KnownPeers is what the lookup asks of the store of found chats and owners.
type KnownPeers interface {
	FindByRef(ctx context.Context, role PeerRole, ref string) (KnownPeer, bool, error)
	Save(ctx context.Context, peer KnownPeer) error
	DeleteByRef(ctx context.Context, role PeerRole, ref string) error
}

// LookupCounts tells how many watched chats came from the store and how many
// were looked up by name.
type LookupCounts struct {
	Stored   int
	LookedUp int
}

// Lookup finds the watched chats and the owner, the store first, waiting
// whenever Telegram asks to. Known nil means nothing is stored.
type Lookup struct {
	Client  Client
	Waiting func(time.Duration)
	Sleep   func(context.Context, time.Duration) error
	Known   KnownPeers
}

// Find finds every chat and the owner and returns the found chats by their
// config text; it returns every problem at once.
func (l *Lookup) Find(ctx context.Context, chats []string, owner string) (map[string]ChatInfo, LookupCounts, error) {
	found, missing, counts, err := l.chats(ctx, chats)
	if err != nil {
		return nil, LookupCounts{}, err
	}
	var chatsErr, ownerErr error
	if len(missing) > 0 {
		chatsErr = &ChatsNotFoundError{Chats: missing}
	}
	ownerState, err := l.owner(ctx, owner)
	if err != nil {
		return nil, LookupCounts{}, err
	}
	switch ownerState {
	case OwnerMissing:
		ownerErr = &OwnerNotFoundError{Owner: owner}
	case OwnerNoChat:
		ownerErr = &OwnerChatClosedError{Owner: owner}
	case OwnerReady:
	}
	if err := errors.Join(chatsErr, ownerErr); err != nil {
		return nil, LookupCounts{}, err
	}
	return found, counts, nil
}

// chats finds every chat: the found ones by their config text, the missing
// ones with their reason, and how many came from the store.
func (l *Lookup) chats(ctx context.Context, chats []string) (map[string]ChatInfo, []MissingChat, LookupCounts, error) {
	var missing []MissingChat
	var counts LookupCounts
	found := map[string]ChatInfo{}
	for _, chat := range chats {
		info, stored, err := l.chat(ctx, chat)
		if err != nil {
			return nil, nil, LookupCounts{}, err
		}
		switch info.State {
		case ChatMissing:
			missing = append(missing, MissingChat{Chat: chat, Reason: "no such chat"})
		case ChatNotJoined:
			missing = append(missing, MissingChat{Chat: chat, Reason: "the account has not joined it"})
		case ChatJoined:
			found[chat] = info
			if stored {
				counts.Stored++
			} else {
				counts.LookedUp++
			}
		}
	}
	return found, missing, counts, nil
}

// chat takes a stored chat, or looks it up by name and saves it when joined;
// stored tells which.
func (l *Lookup) chat(ctx context.Context, chat string) (ChatInfo, bool, error) {
	known, ok, err := l.find(ctx, RoleChat, chat)
	if err != nil {
		return ChatInfo{}, false, err
	}
	if ok {
		info := ChatInfo{State: ChatJoined, ID: known.Peer.ID, Username: known.Username, Title: known.Title,
			Basic: known.Peer.Kind == PeerBasicGroup, Peer: known.Peer}
		l.Client.KeepChat(info)
		return info, true, nil
	}
	info, err := Retry(ctx, l.Waiting, l.Sleep, func() (ChatInfo, error) { return l.Client.ResolveChat(ctx, chat) })
	if err != nil {
		return ChatInfo{}, false, err
	}
	if info.State == ChatJoined {
		err = l.save(ctx, KnownPeer{Role: RoleChat, Ref: chat, Peer: info.Peer, Username: info.Username, Title: info.Title})
	}
	return info, false, err
}

// owner checks a stored owner; when the check fails it drops the stored
// owner and finds it again by name, saving an owner it found.
func (l *Lookup) owner(ctx context.Context, owner string) (OwnerState, error) {
	known, ok, err := l.find(ctx, RoleOwner, owner)
	if err != nil {
		return OwnerMissing, err
	}
	if ok {
		state, err := Retry(ctx, l.Waiting, l.Sleep, func() (OwnerState, error) {
			return l.Client.CheckOwner(ctx, owner, known.Peer)
		})
		if err != nil || state == OwnerReady {
			return state, err
		}
		if err := l.Known.DeleteByRef(ctx, RoleOwner, owner); err != nil {
			return OwnerMissing, fmt.Errorf("stored owner: %w", err)
		}
	}
	info, err := Retry(ctx, l.Waiting, l.Sleep, func() (OwnerInfo, error) { return l.Client.FindOwner(ctx, owner) })
	if err != nil {
		return OwnerMissing, err
	}
	if info.State != OwnerMissing {
		err = l.save(ctx, KnownPeer{Role: RoleOwner, Ref: owner, Peer: info.Peer})
	}
	return info.State, err
}

// Refresh drops a stored chat Telegram no longer knows and looks it up by
// name once; it saves and returns the chat when it is joined under the same
// ID, else it gives ErrChatGone and the next start looks the chat up by name.
func (l *Lookup) Refresh(ctx context.Context, ref string, id int64) (ChatInfo, error) {
	if l.Known != nil {
		if err := l.Known.DeleteByRef(ctx, RoleChat, ref); err != nil {
			return ChatInfo{}, fmt.Errorf("stored chat %s: %w", ref, err)
		}
	}
	info, err := Retry(ctx, l.Waiting, l.Sleep, func() (ChatInfo, error) { return l.Client.ResolveChat(ctx, ref) })
	if err != nil {
		return ChatInfo{}, err
	}
	if info.State != ChatJoined || info.ID != id {
		return ChatInfo{}, ErrChatGone
	}
	if err := l.save(ctx, KnownPeer{Role: RoleChat, Ref: ref, Peer: info.Peer, Username: info.Username,
		Title: info.Title}); err != nil {
		return ChatInfo{}, err
	}
	return info, nil
}

func (l *Lookup) find(ctx context.Context, role PeerRole, ref string) (KnownPeer, bool, error) {
	if l.Known == nil {
		return KnownPeer{}, false, nil
	}
	known, ok, err := l.Known.FindByRef(ctx, role, ref)
	if err != nil {
		return KnownPeer{}, false, fmt.Errorf("stored %s %s: %w", role, ref, err)
	}
	return known, ok, nil
}

func (l *Lookup) save(ctx context.Context, peer KnownPeer) error {
	if l.Known == nil {
		return nil
	}
	if err := l.Known.Save(ctx, peer); err != nil {
		return fmt.Errorf("store %s %s: %w", peer.Role, peer.Ref, err)
	}
	return nil
}

// Retry repeats call after every FLOOD_WAIT, reporting the wait to waiting
// and sleeping it, with no upper limit; any other result is returned.
func Retry[T any](ctx context.Context, waiting func(time.Duration),
	sleep func(context.Context, time.Duration) error, call func() (T, error)) (T, error) {
	for {
		value, err := call()
		var flood *FloodWait
		if !errors.As(err, &flood) {
			return value, err
		}
		waiting(flood.Wait)
		if err := sleep(ctx, flood.Wait); err != nil {
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
