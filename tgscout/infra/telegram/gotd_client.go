package telegram

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/query/dialogs"
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
type gotdClient struct{ client *telegram.Client }

func newGotdClient(cfg Config) *gotdClient {
	return &gotdClient{client: telegram.NewClient(cfg.APIID, cfg.APIHash,
		telegram.Options{SessionStorage: cfg.Storage, NoUpdates: true})}
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

func (c *gotdClient) ResolveChat(ctx context.Context, ref string) (ChatState, error) {
	if !strings.HasPrefix(ref, "@") && !strings.Contains(ref, "t.me/") {
		return c.dialogState(ctx, ref)
	}
	resolved, err := c.resolve(ctx, ref)
	if err != nil || resolved == nil {
		return ChatMissing, err
	}
	for _, chat := range resolved.Chats {
		switch chat := chat.(type) {
		case *tg.Channel:
			return joinedState(chat.Left), nil
		case *tg.Chat:
			return joinedState(chat.Left), nil
		}
	}
	return ChatMissing, nil
}

func (c *gotdClient) FindOwner(ctx context.Context, owner string) (bool, error) {
	if strings.HasPrefix(owner, "@") {
		resolved, err := c.resolve(ctx, owner)
		return resolved != nil && len(resolved.Users) > 0, err
	}
	id, err := strconv.ParseInt(owner, 10, 64)
	if err != nil {
		return false, nil
	}
	contacts, err := c.client.API().ContactsGetContacts(ctx, 0)
	if err != nil {
		return false, mapError(err)
	}
	list, ok := contacts.(*tg.ContactsContacts)
	if !ok {
		return false, nil
	}
	for _, user := range list.Users {
		if user, ok := user.(*tg.User); ok && user.ID == id {
			return true, nil
		}
	}
	return false, nil
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
// the account is in is one of them.
func (c *gotdClient) dialogState(ctx context.Context, ref string) (ChatState, error) {
	want := strings.TrimPrefix(strings.TrimPrefix(ref, "-100"), "-")
	errFound := errors.New("found")
	err := dialogs.NewQueryBuilder(c.client.API()).GetDialogs().BatchSize(100).ForEach(ctx,
		func(_ context.Context, elem dialogs.Elem) error {
			if peerID(elem.Peer) == want {
				return errFound
			}
			return nil
		})
	if errors.Is(err, errFound) {
		return ChatJoined, nil
	}
	return ChatMissing, mapError(err)
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
