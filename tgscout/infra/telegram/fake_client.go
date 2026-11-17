//go:build telegramfake

package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"time"
)

// fakeScenarioEnv names the JSON file of the made-up Telegram the E2E build uses.
const fakeScenarioEnv = "TGSCOUT_FAKE_TELEGRAM"

// fakeScenario is a made-up Telegram: the account, its chats and contacts.
type fakeScenario struct {
	SessionValid     bool              `json:"session_valid"`
	Account          string            `json:"account"`
	Code             string            `json:"code"`
	Password         string            `json:"password"`
	Chats            map[string]string `json:"chats"`
	Contacts         []string          `json:"contacts"`
	FloodWaitSeconds int               `json:"flood_wait_seconds"`
}

func parseFakeScenario(data []byte) (fakeScenario, error) {
	var scenario fakeScenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return fakeScenario{}, fmt.Errorf("fake Telegram scenario: %w", err)
	}
	return scenario, nil
}

// state is the chat's state; the "*" entry stands for every chat not named.
func (s fakeScenario) state(ref string) ChatState {
	state, ok := s.Chats[ref]
	if !ok {
		state = s.Chats["*"]
	}
	switch state {
	case "joined":
		return ChatJoined
	case "not_joined":
		return ChatNotJoined
	}
	return ChatMissing
}

func (s fakeScenario) floodWait() time.Duration {
	return time.Duration(s.FloodWaitSeconds) * time.Second
}

// fakeClient plays the scenario; it stores a made-up session after a sign-in.
type fakeClient struct {
	scenario fakeScenario
	storage  *FileStorage
	flooded  bool
}

// NewClient builds the made-up Telegram of the E2E build.
func NewClient(cfg Config) (Client, error) {
	data, err := os.ReadFile(os.Getenv(fakeScenarioEnv)) //nolint:gosec // G703: the E2E build reads the file its test names
	if err != nil {
		return nil, fmt.Errorf("fake Telegram scenario: %w", err)
	}
	scenario, err := parseFakeScenario(data)
	return &fakeClient{scenario: scenario, storage: cfg.Storage}, err
}

func (c *fakeClient) Run(ctx context.Context, f func(context.Context) error) error { return f(ctx) }

func (c *fakeClient) Authorized(ctx context.Context) (bool, error) {
	_, err := c.storage.LoadSession(ctx)
	return err == nil && c.scenario.SessionValid, nil
}

func (c *fakeClient) SendCode(context.Context, string) (string, error) { return "fake-hash", nil }

func (c *fakeClient) SignIn(ctx context.Context, _, code, _ string) error {
	if code != c.scenario.Code {
		return ErrWrongCode
	}
	if c.scenario.Password != "" {
		return ErrPasswordNeeded
	}
	return c.signedIn(ctx)
}

func (c *fakeClient) CheckPassword(ctx context.Context, password string) error {
	if password != c.scenario.Password {
		return ErrWrongPassword
	}
	return c.signedIn(ctx)
}

func (c *fakeClient) signedIn(ctx context.Context) error {
	c.scenario.SessionValid = true
	return c.storage.StoreSession(ctx, []byte("made-up fake session"))
}

func (c *fakeClient) Self(context.Context) (Account, error) {
	if c.scenario.Account == "" {
		return Account{}, errors.New("fake Telegram scenario: no account")
	}
	return Account{Name: c.scenario.Account}, nil
}

func (c *fakeClient) ResolveChat(_ context.Context, ref string) (ChatState, error) {
	if wait := c.scenario.floodWait(); wait > 0 && !c.flooded {
		c.flooded = true
		return ChatMissing, &FloodWait{Wait: wait}
	}
	return c.scenario.state(ref), nil
}

func (c *fakeClient) FindOwner(_ context.Context, owner string) (bool, error) {
	return slices.Contains(c.scenario.Contacts, owner), nil
}
