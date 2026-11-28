package telegram

import (
	"context"
	"errors"
	"strings"
	"time"
)

// stubClient is a made-up Telegram for the walks' unit tests.
type stubClient struct {
	authorized bool
	authErr    error
	code       string
	password   string
	signedIn   bool
	chats      map[string]ChatState
	owners     map[string]OwnerState
	sent       []string
	floods     map[string]time.Duration
	calls      []string
	updates    []Update
	// resolved and ownersFound list the name lookups; kept the stored chats
	// handed back; ownerChecks the state CheckOwner gives per owner.
	resolved    []string
	ownersFound []string
	kept        []ChatInfo
	ownerChecks map[string]OwnerState
}

func (c *stubClient) Run(ctx context.Context, f func(context.Context) error) error { return f(ctx) }

func (c *stubClient) Authorized(context.Context) (bool, error) { return c.authorized, c.authErr }

func (c *stubClient) SendCode(_ context.Context, phone string) (string, error) {
	c.calls = append(c.calls, "send code "+phone)
	return "made-up-hash", nil
}

func (c *stubClient) SignIn(_ context.Context, _, code, hash string) error {
	if hash != "made-up-hash" || code != c.code {
		return ErrWrongCode
	}
	if c.password != "" {
		return ErrPasswordNeeded
	}
	c.signedIn = true
	return nil
}

func (c *stubClient) CheckPassword(_ context.Context, password string) error {
	if password != c.password {
		return ErrWrongPassword
	}
	c.signedIn = true
	return nil
}

func (c *stubClient) Self(context.Context) (Account, error) {
	return Account{Name: "Made Up Scout"}, nil
}

func (c *stubClient) ResolveChat(_ context.Context, ref string) (ChatInfo, error) {
	c.resolved = append(c.resolved, ref)
	if wait, ok := c.floods[ref]; ok {
		delete(c.floods, ref)
		return ChatInfo{}, &FloodWait{Wait: wait}
	}
	state, ok := c.chats[ref]
	if !ok {
		return ChatInfo{}, nil
	}
	if strings.HasPrefix(ref, "@") {
		return ChatInfo{State: state, ID: 7, Username: ref[1:], Title: "Made-up title of " + ref[1:],
			Peer: Peer{Kind: PeerChannel, ID: 7, AccessHash: 70}}, nil
	}
	return ChatInfo{State: state, ID: bareChatID(ref), Peer: Peer{Kind: PeerChannel, ID: bareChatID(ref), AccessHash: 71}}, nil
}

func (c *stubClient) KeepChat(info ChatInfo) { c.kept = append(c.kept, info) }

func (c *stubClient) CheckOwner(_ context.Context, owner string, _ Peer) (OwnerState, error) {
	return c.ownerChecks[owner], nil
}

func (c *stubClient) Listen(ctx context.Context, f func(context.Context, Update)) error {
	for _, update := range c.updates {
		f(ctx, update)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (c *stubClient) ReadHistory(context.Context, int64, HistoryWindow) (History, error) {
	return History{}, nil
}

func (c *stubClient) FindOwner(_ context.Context, owner string) (OwnerInfo, error) {
	c.ownersFound = append(c.ownersFound, owner)
	state := c.owners[owner]
	if state == OwnerMissing {
		return OwnerInfo{}, nil
	}
	return OwnerInfo{State: state, Peer: Peer{Kind: PeerUser, ID: 42, AccessHash: 4242}}, nil
}

func (c *stubClient) SendToOwner(_ context.Context, owner, text string) error {
	c.sent = append(c.sent, owner+": "+text)
	return nil
}

// scriptedPrompter answers from fixed lists and records what it was told.
type scriptedPrompter struct {
	phones, codes, passwords []string
	notices                  []Notice
}

func (p *scriptedPrompter) Phone(context.Context) (string, error) { return next(&p.phones) }
func (p *scriptedPrompter) Code(context.Context) (string, error)  { return next(&p.codes) }
func (p *scriptedPrompter) Password(context.Context) (string, error) {
	return next(&p.passwords)
}
func (p *scriptedPrompter) Notice(n Notice) { p.notices = append(p.notices, n) }

func next(list *[]string) (string, error) {
	if len(*list) == 0 {
		return "", errors.New("no more input")
	}
	value := (*list)[0]
	*list = (*list)[1:]
	return value, nil
}

// stubSession records the commit.
type stubSession struct {
	firstRun  bool
	committed bool
}

func (s *stubSession) FirstRun() bool               { return s.firstRun }
func (s *stubSession) Commit(context.Context) error { s.committed = true; return nil }

func (c *stubClient) WatchConnection(func(ConnectionEvent)) {}
