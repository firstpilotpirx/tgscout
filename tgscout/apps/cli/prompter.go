package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// consolePrompter asks for the phone and the code on the console, visibly,
// and for the password through readPassword, hidden.
type consolePrompter struct {
	con          *console
	in           *bufio.Reader
	readPassword func(context.Context) (string, error)
}

func newConsolePrompter(con *console, in io.Reader, readPassword func(context.Context) (string, error)) *consolePrompter {
	return &consolePrompter{con: con, in: bufio.NewReader(in), readPassword: readPassword}
}

func (p *consolePrompter) Phone(ctx context.Context) (string, error) {
	return p.ask(ctx, "Phone number: ")
}

func (p *consolePrompter) Code(ctx context.Context) (string, error) {
	return p.ask(ctx, "Login code: ")
}

func (p *consolePrompter) Password(ctx context.Context) (string, error) {
	p.con.print("Two-step password: ")
	password, err := p.readPassword(ctx)
	p.con.println(p.con.out, "") // the hidden input leaves the cursor on the prompt line
	return password, err
}

func (p *consolePrompter) Notice(n telegram.Notice) {
	switch n {
	case telegram.NoticeFirstSignIn:
		p.con.firstSignIn()
	case telegram.NoticeCodeSent:
		p.con.codeSent()
	case telegram.NoticeWrongCode:
		p.con.wrongCode()
	case telegram.NoticeWrongPassword:
		p.con.wrongPassword()
	}
}

func (p *consolePrompter) ask(ctx context.Context, prompt string) (string, error) {
	p.con.print(prompt)
	return readCancellable(ctx, func() (string, error) {
		line, err := p.in.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimSpace(line), nil
	})
}

// readCancellable stops waiting for input when ctx ends (Ctrl+C at a prompt).
func readCancellable(ctx context.Context, read func() (string, error)) (string, error) {
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		text, err := read()
		done <- result{text, err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-done:
		return r.text, r.err
	}
}

// readHiddenPassword reads the password from the terminal without echo.
func readHiddenPassword(ctx context.Context) (string, error) {
	return readCancellable(ctx, func() (string, error) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		return string(password), err
	})
}

// stdinIsTerminal tells whether a first sign-in may ask for input.
func stdinIsTerminal() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
