package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// startDeps are the parts of the start that touch the world outside the
// config and the database; tests swap them.
type startDeps struct {
	getenv       func(string) (string, bool)
	readDotEnv   func() string
	isTerminal   func() bool
	stdin        io.Reader
	readPassword func(context.Context) (string, error)
	newClient    func(telegram.Config) (telegram.Client, error)
	sleep        func(context.Context, time.Duration) error
}

func defaultDeps() startDeps {
	return startDeps{
		getenv: os.LookupEnv,
		readDotEnv: func() string {
			data, err := os.ReadFile(dotEnvFile)
			if err != nil {
				return "" // no .env: the environment alone holds the variables
			}
			return string(data)
		},
		isTerminal:   stdinIsTerminal,
		stdin:        os.Stdin,
		readPassword: readHiddenPassword,
		newClient:    telegram.NewClient,
		sleep:        telegram.SleepContext,
	}
}

// watch signs in, finds the chats and the owner, then matches their new
// messages until ctx ends; the workers and the listener stop before it returns.
// known keeps the found chats and the owner for the next start.
func watch(ctx context.Context, con *console, deps startDeps, config string, matcher live,
	known telegram.KnownPeers) int {
	settings := matcher.settings
	link, ok := openTelegram(con, deps, config, settings.HistoryPause)
	if !ok {
		return exitError
	}
	link.known = known
	err := runDetached(ctx, link.client, func(ctx context.Context) error {
		found, err := connect(ctx, link, con, watchedChats(settings), settings.Owner.String())
		if err != nil {
			return err
		}
		return matcher.serve(ctx, con, link.client, found)
	})
	switch {
	case errors.Is(err, errReported):
		return exitError
	case err != nil && ctx.Err() == nil:
		con.fail("cannot reach Telegram", "check the network, then start tgscout again")
		return exitError
	}
	return exitOK
}

// openTelegram reads the credentials, checks the session file and makes the
// client, whose history calls keep historyPause between them.
func openTelegram(con *console, deps startDeps, config string, historyPause time.Duration) (telegramLink, bool) {
	creds, err := loadCredentials(deps.getenv, parseDotEnv(deps.readDotEnv()), config)
	if err != nil {
		con.fail(err.Error(), "fill it in .env (see .env.example)")
		return telegramLink{}, false
	}
	exists, ok := checkSession(con, creds.sessionFile)
	if !ok {
		return telegramLink{}, false
	}
	if !exists && !deps.isTerminal() {
		con.fail("first sign-in needs a terminal", "sign in once from a terminal first, then run it as a service")
		return telegramLink{}, false
	}
	storage := telegram.NewFileStorage(creds.sessionFile, !exists)
	client, err := deps.newClient(telegram.Config{APIID: creds.apiID, APIHash: creds.apiHash, Storage: storage,
		HistoryPause: historyPause})
	if err != nil {
		con.fail("cannot start the Telegram client: "+err.Error(), "")
		return telegramLink{}, false
	}
	return telegramLink{
		client:      client,
		prompter:    newConsolePrompter(con, deps.stdin, deps.readPassword),
		session:     storage,
		sessionPath: creds.sessionFile,
		sleep:       deps.sleep,
	}, true
}

// checkSession tells whether the session file exists; an open one stops the start.
func checkSession(con *console, path string) (bool, bool) {
	exists, err := telegram.CheckSessionFile(path)
	var open *telegram.SessionOpenError
	switch {
	case errors.As(err, &open):
		con.fail("the session file is open to other users",
			fmt.Sprintf("%s has mode %o · Fix: chmod 600 %s", open.Path, open.Mode.Perm(), open.Path), open.Path)
		return false, false
	case err != nil:
		con.fail(fmt.Sprintf("session file %s: %v", path, err), "", path)
		return false, false
	}
	return exists, true
}

// watchedChats lists every chat of every request once, in config order.
func watchedChats(settings domain.Settings) []string {
	seen := map[string]bool{}
	var chats []string
	for _, request := range settings.Requests {
		for _, chat := range request.Chats {
			if !seen[chat.Ref.Key()] {
				seen[chat.Ref.Key()] = true
				chats = append(chats, chat.Ref.String())
			}
		}
	}
	return chats
}
