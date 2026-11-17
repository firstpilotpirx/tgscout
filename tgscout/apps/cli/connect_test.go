package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// fakeTelegram is a made-up Telegram for the start step: no network.
type fakeTelegram struct {
	authorized bool
	code       string
	password   string
	chats      map[string]telegram.ChatState
	owners     []string
	everyChat  bool // every chat counts as joined
}

func (f *fakeTelegram) Run(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
func (f *fakeTelegram) Authorized(context.Context) (bool, error)                      { return f.authorized, nil }
func (f *fakeTelegram) SendCode(context.Context, string) (string, error)              { return "hash", nil }

func (f *fakeTelegram) SignIn(_ context.Context, _, code, _ string) error {
	if code != f.code {
		return telegram.ErrWrongCode
	}
	return telegram.ErrPasswordNeeded
}

func (f *fakeTelegram) CheckPassword(_ context.Context, password string) error {
	if password != f.password {
		return telegram.ErrWrongPassword
	}
	f.authorized = true
	return nil
}

func (f *fakeTelegram) Self(context.Context) (telegram.Account, error) {
	return telegram.Account{Name: "Made Up Reader"}, nil
}

func (f *fakeTelegram) ResolveChat(_ context.Context, ref string) (telegram.ChatState, error) {
	if f.everyChat {
		return telegram.ChatJoined, nil
	}
	return f.chats[ref], nil
}

func (f *fakeTelegram) FindOwner(_ context.Context, owner string) (bool, error) {
	for _, each := range f.owners {
		if each == owner {
			return true, nil
		}
	}
	return false, nil
}

// heldSession records whether the sign-in saved it.
type heldSession struct {
	firstRun  bool
	committed bool
}

func (s *heldSession) FirstRun() bool               { return s.firstRun }
func (s *heldSession) Commit(context.Context) error { s.committed = true; return nil }

func everythingFound() *fakeTelegram {
	return &fakeTelegram{
		code: "12345", password: "made-up-password",
		chats:  map[string]telegram.ChatState{"@made_up_market": telegram.ChatJoined, "@made_up_river": telegram.ChatJoined},
		owners: []string{"@made_up_owner"},
	}
}

type connectRun struct {
	out, errOut bytes.Buffer
	session     heldSession
	err         error
}

func runConnect(t *testing.T, client *fakeTelegram, firstRun bool, typed string, chats ...string) *connectRun {
	t.Helper()
	r := &connectRun{session: heldSession{firstRun: firstRun}}
	con := newConsole(&r.out, &r.errOut, true)
	passwords := strings.NewReader(client.password + "\n")
	link := telegramLink{
		client:      client,
		prompter:    newConsolePrompter(con, strings.NewReader(typed), func(context.Context) (string, error) { return readLine(passwords) }),
		session:     &r.session,
		sessionPath: "/made/up/tgscout.session",
		sleep:       func(context.Context, time.Duration) error { return nil },
	}
	r.err = connect(context.Background(), link, con, chats, "@made_up_owner")
	return r
}

func readLine(r *strings.Reader) (string, error) {
	var line strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil || b == '\n' {
			return line.String(), err
		}
		line.WriteByte(b)
	}
}

func TestConnectFirstRunPrintsTheSignInLinesInOrder(t *testing.T) {
	r := runConnect(t, everythingFound(), true, "+00000000000\n12345\n", "@made_up_market", "@made_up_river")

	require.NoError(t, r.err, r.errOut.String())
	assert.True(t, r.session.committed)
	assert.Equal(t, "[run] no session yet — first sign-in\n"+
		"Phone number: Telegram sent a login code to the account's app.\n"+
		"Login code: Two-step password: \n"+
		"[ok] signed in as Made Up Reader\n"+
		"[ok] session saved /made/up/tgscout.session (mode 600)\n"+
		"[ok] owner found: @made_up_owner\n"+
		"[ok] watching 2 chats\n", r.out.String())
}

func TestConnectWrongCodeAsksAgain(t *testing.T) {
	r := runConnect(t, everythingFound(), true, "+00000000000\n99999\n12345\n", "@made_up_market")

	require.NoError(t, r.err)
	assert.Equal(t, "[fail] the code is wrong — try again\n", r.errOut.String())
	assert.Equal(t, 2, strings.Count(r.out.String(), "Login code: "))
}

func TestConnectSavedSessionAsksNothing(t *testing.T) {
	client := everythingFound()
	client.authorized = true

	r := runConnect(t, client, false, "", "@made_up_market")

	require.NoError(t, r.err)
	assert.False(t, r.session.committed)
	assert.Equal(t, "[ok] signed in as Made Up Reader (saved session)\n"+
		"[ok] owner found: @made_up_owner\n[ok] watching 1 chat\n", r.out.String())
}

func TestConnectRevokedSessionSaysToSignInAgain(t *testing.T) {
	r := runConnect(t, everythingFound(), false, "", "@made_up_market")

	require.Error(t, r.err)
	assert.Equal(t, "[fail] Telegram no longer accepts the session\n"+
		"  delete /made/up/tgscout.session, then start tgscout from a terminal\n", r.errOut.String())
}

func TestConnectLookupFailureListsEveryProblem(t *testing.T) {
	client := everythingFound()
	client.authorized = true
	client.chats["@made_up_river"] = telegram.ChatNotJoined
	client.owners = nil

	r := runConnect(t, client, false, "", "@made_up_market", "@made_up_gone", "@made_up_river")

	require.Error(t, r.err)
	assert.Equal(t, "[fail] 2 chats not found\n"+
		"  @made_up_gone   no such chat\n"+
		"  @made_up_river  the account has not joined it\n"+
		"  join them with tgscout's account, or remove them from tgscout.yaml\n"+
		"[fail] owner not found: @made_up_owner\n"+
		"  add the owner to the contacts of tgscout's account (README → Setup)\n", r.errOut.String())
	assert.NotContains(t, r.out.String(), "watching")
}

// testDeps give every start made-up credentials and a Telegram where the
// session is valid and every chat and the owner are found.
func testDeps() startDeps {
	env := map[string]string{envAPIID: "12345", envAPIHash: "made-up-hash"}
	return startDeps{
		getenv:       func(key string) (string, bool) { value, ok := env[key]; return value, ok },
		readDotEnv:   func() string { return "" },
		isTerminal:   func() bool { return true },
		stdin:        strings.NewReader(""),
		readPassword: func(context.Context) (string, error) { return "", nil },
		newClient: func(telegram.Config) (telegram.Client, error) {
			client := everythingFound()
			client.authorized, client.everyChat = true, true
			return client, nil
		},
		sleep: func(context.Context, time.Duration) error { return nil },
	}
}
