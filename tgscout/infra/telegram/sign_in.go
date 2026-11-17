package telegram

import (
	"context"
	"errors"
	"fmt"
)

// Notice is a sign-in event the console shows.
type Notice int

// The sign-in events.
const (
	NoticeFirstSignIn Notice = iota
	NoticeCodeSent
	NoticeWrongCode
	NoticeWrongPassword
)

// Prompter asks the owner for what a first sign-in needs.
type Prompter interface {
	Phone(ctx context.Context) (string, error)
	Code(ctx context.Context) (string, error)
	Password(ctx context.Context) (string, error)
	Notice(n Notice)
}

// Session is the session store as the sign-in sees it.
type Session interface {
	FirstRun() bool
	// Commit saves the session once the sign-in succeeded.
	Commit(ctx context.Context) error
}

// SignIn signs in from the saved session, or on a first run asks for the
// phone, the code and the password, and saves the session only after success.
func SignIn(ctx context.Context, client Client, prompter Prompter, session Session) (Account, error) {
	authorized, err := client.Authorized(ctx)
	if err != nil {
		return Account{}, err
	}
	if !authorized {
		if !session.FirstRun() {
			return Account{}, ErrSessionRevoked
		}
		if err := firstSignIn(ctx, client, prompter); err != nil {
			return Account{}, err
		}
		if err := session.Commit(ctx); err != nil {
			return Account{}, fmt.Errorf("save the session: %w", err)
		}
	}
	return client.Self(ctx)
}

func firstSignIn(ctx context.Context, client Client, prompter Prompter) error {
	prompter.Notice(NoticeFirstSignIn)
	phone, err := prompter.Phone(ctx)
	if err != nil {
		return err
	}
	hash, err := client.SendCode(ctx, phone)
	if err != nil {
		return err
	}
	prompter.Notice(NoticeCodeSent)
	for {
		code, err := prompter.Code(ctx)
		if err != nil {
			return err
		}
		err = client.SignIn(ctx, phone, code, hash)
		switch {
		case errors.Is(err, ErrWrongCode):
			prompter.Notice(NoticeWrongCode)
		case errors.Is(err, ErrPasswordNeeded):
			return checkPassword(ctx, client, prompter)
		default:
			return err
		}
	}
}

func checkPassword(ctx context.Context, client Client, prompter Prompter) error {
	for {
		password, err := prompter.Password(ctx)
		if err != nil {
			return err
		}
		err = client.CheckPassword(ctx, password)
		if !errors.Is(err, ErrWrongPassword) {
			return err
		}
		prompter.Notice(NoticeWrongPassword)
	}
}
