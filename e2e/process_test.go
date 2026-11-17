package e2e_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const waitLimit = 10 * time.Second

// syncBuffer collects the output of the process while it runs.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// process is one run of the built tgscout binary.
type process struct {
	cmd      *exec.Cmd
	stdout   syncBuffer
	stderr   syncBuffer
	done     chan struct{}
	code     int
	terminal *os.File // the pseudo-terminal of a run on one, else nil
	tty      *os.File // its terminal side, kept to read the echo state
}

// newProcess prepares the binary named by SUT_BIN in dir, without color.
func newProcess(dir string, env []string, args ...string) (*process, error) {
	bin := os.Getenv("SUT_BIN")
	if bin == "" {
		return nil, errors.New("SUT_BIN must name the built tgscout binary")
	}
	p := &process{cmd: exec.Command(bin, args...), done: make(chan struct{})} //nolint:gosec // G204: the binary under test, named by SUT_BIN
	p.cmd.Dir = dir
	p.cmd.Env = append(append(cleanEnviron(), "NO_COLOR=1"), env...)
	return p, nil
}

// startProcess runs the binary with its input from /dev/null, not a terminal.
func startProcess(dir string, env []string, args ...string) (*process, error) {
	p, err := newProcess(dir, env, args...)
	if err != nil {
		return nil, err
	}
	p.cmd.Stdout = &p.stdout
	p.cmd.Stderr = &p.stderr
	if err := p.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", p.cmd.Path, err)
	}
	go p.wait()
	return p, nil
}

func (p *process) wait() {
	err := p.cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		p.code = exitErr.ExitCode()
	}
	close(p.done)
}

// settle waits until the process runs (it printed the running line) or exited.
func (p *process) settle() error {
	deadline := time.After(waitLimit)
	for {
		select {
		case <-p.done:
			return nil
		case <-deadline:
			return fmt.Errorf("tgscout neither ran nor exited in %s; output: %s", waitLimit, p.output())
		case <-time.After(20 * time.Millisecond):
			if strings.Contains(p.stdout.String(), "running") {
				return nil
			}
		}
	}
}

// signal sends sig and waits for the exit.
func (p *process) signal(sig syscall.Signal) error {
	if err := p.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("send %s: %w", sig, err)
	}
	return p.exited()
}

// send sends sig and does not wait.
func (p *process) send(sig syscall.Signal) error {
	if err := p.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("send %s: %w", sig, err)
	}
	return nil
}

// exitedWithin waits for the exit and says how long it took.
func (p *process) exitedWithin() (time.Duration, error) {
	started := time.Now()
	err := p.exited()
	return time.Since(started), err
}

func (p *process) exited() error {
	select {
	case <-p.done:
		return nil
	case <-time.After(waitLimit):
		return fmt.Errorf("tgscout did not exit in %s; output: %s", waitLimit, p.output())
	}
}

func (p *process) stop() {
	select {
	case <-p.done:
	default:
		_ = p.cmd.Process.Kill() // cleanup after a scenario; the exit is not checked
		<-p.done
	}
	p.closeTerminal()
}

func (p *process) output() string { return p.stdout.String() + p.stderr.String() }

// cleanEnviron is the runner's environment without tgscout's own variables,
// so a developer's or CI's Telegram settings never reach a scenario.
func cleanEnviron() []string {
	var env []string
	for _, each := range os.Environ() {
		if !strings.HasPrefix(each, "TG_") && !strings.HasPrefix(each, "TGSCOUT_") {
			env = append(env, each)
		}
	}
	return env
}
