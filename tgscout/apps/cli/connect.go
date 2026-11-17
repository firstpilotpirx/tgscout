package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// errReported is a start error already printed on the console.
var errReported = errors.New("reported")

// telegramLink is what the sign-in step works with.
type telegramLink struct {
	client      telegram.Client
	prompter    telegram.Prompter
	session     telegram.Session
	sessionPath string
	sleep       func(context.Context, time.Duration) error
}

// connect signs in and finds the watched chats and the owner, printing each
// step; on a problem it prints it and returns errReported.
func connect(ctx context.Context, link telegramLink, con *console, chats []string, owner string) error {
	firstRun := link.session.FirstRun()
	account, err := telegram.SignIn(ctx, link.client, link.prompter, link.session)
	if err != nil {
		return reportSignIn(con, err, link.sessionPath)
	}
	con.signedIn(account.Name, !firstRun)
	if firstRun {
		con.sessionSaved(link.sessionPath)
	}
	lookup := telegram.Lookup{Client: link.client, Waiting: con.floodWait, Sleep: link.sleep}
	if err := lookup.Find(ctx, chats, owner); err != nil {
		return reportLookup(con, err)
	}
	con.ownerFound(owner)
	con.watching(len(chats))
	return nil
}

func reportSignIn(con *console, err error, path string) error {
	switch {
	case isStop(err):
		return err
	case errors.Is(err, telegram.ErrSessionRevoked):
		con.fail("Telegram no longer accepts the session", fmt.Sprintf("delete %s, then start tgscout from a terminal", path))
	default:
		con.fail("cannot sign in: "+err.Error(), "")
	}
	return errReported
}

func reportLookup(con *console, err error) error {
	if isStop(err) {
		return err
	}
	var chatsErr *telegram.ChatsNotFoundError
	var ownerErr *telegram.OwnerNotFoundError
	if errors.As(err, &chatsErr) {
		con.chatsNotFound(chatsErr.Chats)
	}
	if errors.As(err, &ownerErr) {
		con.fail("owner not found: "+ownerErr.Owner, "add the owner to the contacts of tgscout's account (README → Setup)")
	}
	if chatsErr == nil && ownerErr == nil {
		con.fail(err.Error(), "")
	}
	return errReported
}

// isStop tells whether err is the stop (Ctrl+C), not a problem to report.
func isStop(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
