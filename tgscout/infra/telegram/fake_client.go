//go:build telegramfake

package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"slices"
	"strings"
	"sync"
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
	Updates          []fakeUpdate      `json:"updates"`
	Titles           map[string]string `json:"titles"`
	OwnerChatClosed  bool              `json:"owner_chat_closed"`
	SendFails        bool              `json:"send_fails"`
	SendFloodSeconds int               `json:"send_flood_wait_seconds"`
	SentFile         string            `json:"sent_file"`
	History          []fakeHistory     `json:"history"`
	HistoryFlood     int               `json:"history_flood_wait_seconds"`
	HistoryDelayMs   int               `json:"history_delay_ms"`
	SendHangMs       int               `json:"send_hang_ms"`
}

// fakeHistory is a made-up message posted before the start; Chat is the
// config text of a watched chat.
type fakeHistory struct {
	Chat       string `json:"chat"`
	ID         int    `json:"id"`
	Text       string `json:"text"`
	Service    bool   `json:"service"`
	AgeMinutes int    `json:"age_minutes"`
}

// fakeSend is one line of the sent file: whom the fake sent which text.
type fakeSend struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

// fakeUpdate is a made-up message the fake plays after Listen starts; Chat
// is the config text of a watched chat.
type fakeUpdate struct {
	Chat    string `json:"chat"`
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Service bool   `json:"service"`
	Edited  bool   `json:"edited"`
	Private bool   `json:"private"`
	AfterMs int    `json:"after_ms"`
	// ConnectionLost and ConnectionBack play a drop of the connection and its return.
	ConnectionLost bool `json:"connection_lost"`
	ConnectionBack bool `json:"connection_back"`
	// Lost is a message posted while the connection is away: never delivered
	// live, served by the history from the moment it is played.
	Lost bool `json:"lost"`
}

// info gives a chat its state and a made-up ID: a numeric reference keeps
// its own, a username gets one from its name.
func (s fakeScenario) info(ref string) ChatInfo {
	if !strings.HasPrefix(ref, "@") && !strings.Contains(ref, "t.me/") {
		return ChatInfo{State: s.state(ref), ID: bareChatID(ref), Title: s.Titles[ref]}
	}
	name := strings.ToLower(ref[strings.LastIndexAny(ref, "@/")+1:])
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(name))
	return ChatInfo{State: s.state(ref), ID: 9_000_000_000 + int64(hash.Sum32()), Username: name,
		Title: s.Titles[ref]}
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
	scenario    fakeScenario
	storage     *FileStorage
	flooded     bool
	sendFlooded bool
	sendMu      sync.Mutex
	historyMu   sync.Mutex
	histFlooded bool
	lostHistory []fakeHistory
	watchMu     sync.Mutex
	watcher     func(ConnectionEvent)
}

// WatchConnection keeps f for the drops and returns the scenario plays.
func (c *fakeClient) WatchConnection(f func(ConnectionEvent)) {
	c.watchMu.Lock()
	defer c.watchMu.Unlock()
	c.watcher = f
}

func (c *fakeClient) connection(event ConnectionEvent) {
	c.watchMu.Lock()
	f := c.watcher
	c.watchMu.Unlock()
	if f != nil {
		f(event)
	}
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

func (c *fakeClient) ResolveChat(_ context.Context, ref string) (ChatInfo, error) {
	if wait := c.scenario.floodWait(); wait > 0 && !c.flooded {
		c.flooded = true
		return ChatInfo{}, &FloodWait{Wait: wait}
	}
	return c.scenario.info(ref), nil
}

// Listen plays the scenario's updates in order, then waits for the stop.
func (c *fakeClient) Listen(ctx context.Context, f func(context.Context, Update)) error {
	for _, update := range c.scenario.Updates {
		if err := SleepContext(ctx, time.Duration(update.AfterMs)*time.Millisecond); err != nil {
			return err
		}
		switch {
		case update.ConnectionLost:
			c.connection(ConnectionLost)
			continue
		case update.ConnectionBack:
			c.connection(ConnectionBack)
			continue
		case update.Lost:
			c.historyMu.Lock()
			c.lostHistory = append(c.lostHistory, fakeHistory{Chat: update.Chat, ID: update.ID, Text: update.Text,
				Service: update.Service})
			c.historyMu.Unlock()
			continue
		case update.Private:
			continue // as gotd's mapping: a private chat with a user is never an update
		}
		f(ctx, Update{ChatID: c.scenario.info(update.Chat).ID, MessageID: update.ID, Date: time.Now(),
			Text: update.Text, Service: update.Service, Edited: update.Edited})
	}
	<-ctx.Done()
	return ctx.Err()
}

func (c *fakeClient) FindOwner(_ context.Context, owner string) (OwnerState, error) {
	switch {
	case !slices.Contains(c.scenario.Contacts, owner):
		return OwnerMissing, nil
	case c.scenario.OwnerChatClosed:
		return OwnerNoChat, nil
	}
	return OwnerReady, nil
}

// SendToOwner appends the send to the scenario's sent file; the workers send
// at once, so the sends go one at a time. With send_hang_ms it waits after
// the send, as a confirmation that never comes in time.
func (c *fakeClient) SendToOwner(ctx context.Context, owner, text string) error {
	if err := c.send(owner, text); err != nil {
		return err
	}
	return SleepContext(ctx, time.Duration(c.scenario.SendHangMs)*time.Millisecond)
}

func (c *fakeClient) send(owner, text string) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if wait := time.Duration(c.scenario.SendFloodSeconds) * time.Second; wait > 0 && !c.sendFlooded {
		c.sendFlooded = true
		return &FloodWait{Wait: wait}
	}
	if c.scenario.SendFails {
		return errTelegram
	}
	line, err := json.Marshal(fakeSend{To: owner, Text: text})
	if err != nil {
		return fmt.Errorf("fake Telegram send: %w", err)
	}
	file, err := os.OpenFile(c.scenario.SentFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G703: the E2E build writes the file its test names
	if err != nil {
		return fmt.Errorf("fake Telegram send: %w", err)
	}
	_, err = file.Write(append(line, '\n'))
	return errors.Join(err, file.Close())
}

// ReadHistory pauses history_delay_ms, asks to wait once when the scenario
// says so, then gives the window of the chat's made-up history.
func (c *fakeClient) ReadHistory(ctx context.Context, chatID int64, window HistoryWindow) (History, error) {
	if err := SleepContext(ctx, time.Duration(c.scenario.HistoryDelayMs)*time.Millisecond); err != nil {
		return History{}, err
	}
	c.historyMu.Lock()
	defer c.historyMu.Unlock()
	if wait := time.Duration(c.scenario.HistoryFlood) * time.Second; wait > 0 && !c.histFlooded {
		c.histFlooded = true
		return History{}, &FloodWait{Wait: wait}
	}
	now := time.Now()
	var newestFirst []Update
	for _, each := range slices.Concat(c.scenario.History, c.lostHistory) {
		if c.scenario.info(each.Chat).ID != chatID {
			continue
		}
		newestFirst = append(newestFirst, Update{ChatID: chatID, MessageID: each.ID, Text: each.Text,
			Service: each.Service, Date: now.Add(-time.Duration(each.AgeMinutes) * time.Minute)})
	}
	slices.SortFunc(newestFirst, func(a, b Update) int { return b.MessageID - a.MessageID })
	return cutWindow(newestFirst, window), nil
}
