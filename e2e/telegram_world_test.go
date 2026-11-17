package e2e_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// The made-up secrets of every scenario; the secrets scenario looks for each in the output.
const (
	madeUpAPIID      = "24680"
	madeUpAPIHash    = "made-up-hash-5e6f7a"
	madeUpPhone      = "+10000000001"
	madeUpCode       = "86420"
	madeUpWrongCode  = "13579"
	madeUpPassword   = "made-up-password-7"
	madeUpSession    = "made-up fake session"
	madeUpAccount    = "Made Up Reader"
	fakeScenarioFile = "fake-telegram.json"
	sentFileName     = "fake-telegram-sent.jsonl"
	marketTitle      = "Made-up market"
	sessionFileName  = "tgscout.session"
)

// fakeTelegram is the scenario the telegramfake build reads (TGSCOUT_FAKE_TELEGRAM).
type fakeTelegram struct {
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
	SentFile         string            `json:"sent_file"`
	History          []fakeHistory     `json:"history"`
	HistoryFlood     int               `json:"history_flood_wait_seconds"`
	HistoryDelayMs   int               `json:"history_delay_ms"`
}

// fakeHistory is a made-up message posted before the start, AgeMinutes ago.
type fakeHistory struct {
	Chat       string `json:"chat"`
	ID         int    `json:"id"`
	Text       string `json:"text"`
	AgeMinutes int    `json:"age_minutes"`
}

// fakeUpdate is a made-up message the fake Telegram plays after the start.
type fakeUpdate struct {
	Chat    string `json:"chat"`
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Edited  bool   `json:"edited"`
	Private bool   `json:"private"`
	AfterMs int    `json:"after_ms"`
}

// telegramWorld is the Telegram side of a scenario: by default a valid
// private session where every chat and the owner are found.
type telegramWorld struct {
	fake        fakeTelegram
	session     bool
	sessionMode fs.FileMode
	noAPIHash   bool
}

func defaultTelegram() telegramWorld {
	return telegramWorld{
		fake: fakeTelegram{
			SessionValid: true, Account: madeUpAccount, Code: madeUpCode,
			Chats: map[string]string{"*": "joined"}, Contacts: []string{"@made_up_owner"},
			Titles: map[string]string{"@made_up_market": marketTitle},
		},
		session: true, sessionMode: 0o600,
	}
}

// write puts the fake scenario and the session file into dir and returns the environment.
func (t telegramWorld) write(dir string) ([]string, error) {
	t.fake.SentFile = filepath.Join(dir, sentFileName)
	data, err := json.Marshal(t.fake) //nolint:gosec // G117: a made-up password of the fake Telegram
	if err != nil {
		return nil, fmt.Errorf("fake scenario: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, fakeScenarioFile), data, 0o600); err != nil {
		return nil, fmt.Errorf("fake scenario: %w", err)
	}
	session := filepath.Join(dir, sessionFileName)
	if t.session {
		if err := writeWithMode(session, madeUpSession, t.sessionMode); err != nil {
			return nil, err
		}
	}
	env := []string{"TG_API_ID=" + madeUpAPIID, "TG_SESSION_FILE=" + session,
		"TGSCOUT_FAKE_TELEGRAM=" + filepath.Join(dir, fakeScenarioFile)}
	if !t.noAPIHash {
		env = append(env, "TG_API_HASH="+madeUpAPIHash)
	}
	return env, nil
}

func writeWithMode(path, text string, mode fs.FileMode) error {
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return fmt.Errorf("session file: %w", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("session file: %w", err)
	}
	return nil
}
