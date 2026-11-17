package e2e_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// startOnTerminal runs the binary on a pseudo-terminal, as the owner does at
// a first sign-in. With echo off the output holds only what tgscout itself
// prints; with echo on it also holds what the terminal shows of the typing.
func startOnTerminal(dir string, env []string, echo bool, args ...string) (*process, error) {
	p, err := newProcess(dir, env, args...)
	if err != nil {
		return nil, err
	}
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, fmt.Errorf("open a pseudo-terminal: %w", err)
	}
	if !echo {
		if _, err := term.MakeRaw(int(tty.Fd())); err != nil {
			return nil, errors.Join(fmt.Errorf("turn off the echo: %w", err), ptmx.Close(), tty.Close())
		}
	}
	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = tty, tty, tty
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := p.cmd.Start(); err != nil {
		return nil, errors.Join(fmt.Errorf("start %s: %w", p.cmd.Path, err), ptmx.Close(), tty.Close())
	}
	p.terminal, p.tty = ptmx, tty
	go func() { _, _ = io.Copy(&p.stdout, ptmx) }() // ends with an error when the child exits
	go p.wait()
	return p, nil
}

// waitFor waits until the output shows prompt for the nth time.
func (p *process) waitFor(prompt string, nth int) error {
	deadline := time.Now().Add(waitLimit)
	for strings.Count(p.output(), prompt) < nth {
		if time.Now().After(deadline) {
			return fmt.Errorf("tgscout did not ask %q (%d) in %s; output: %s", prompt, nth, waitLimit, p.output())
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// typeAfter waits for the nth prompt, then types text and Enter.
func (p *process) typeAfter(prompt string, nth int, text string) error {
	if err := p.waitFor(prompt, nth); err != nil {
		return err
	}
	if _, err := io.WriteString(p.terminal, text+"\n"); err != nil {
		return fmt.Errorf("type into tgscout: %w", err)
	}
	return nil
}

// closeTerminal closes the terminal of a run on one.
func (p *process) closeTerminal() {
	if p.terminal != nil {
		_ = p.terminal.Close() // cleanup after a scenario
		_ = p.tty.Close()
	}
}

// waitEchoOff waits until tgscout turned the terminal's echo off, as it does
// for the password, so nothing typed after it can be shown by the terminal.
func (p *process) waitEchoOff() error {
	deadline := time.Now().Add(waitLimit)
	for {
		state, err := unix.IoctlGetTermios(int(p.tty.Fd()), getTermios)
		if err != nil {
			return fmt.Errorf("read the terminal state: %w", err)
		}
		if state.Lflag&unix.ECHO == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("tgscout did not hide the input in %s; output: %s", waitLimit, p.output())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
