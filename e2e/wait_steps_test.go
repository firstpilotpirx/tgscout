package e2e_test

import (
	"fmt"
	"time"
)

// eventually waits until done holds, at most waitLimit.
func (w *world) eventually(what string, done func() bool) error {
	deadline := time.Now().Add(waitLimit)
	for !done() {
		if time.Now().After(deadline) {
			return fmt.Errorf("%s never held in %s; output: %s", what, waitLimit, w.run.output())
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// waitIdle waits until the LLM calls and the console stay the same for idleFor.
func (w *world) waitIdle() error {
	deadline := time.Now().Add(waitLimit)
	seen, since := "", time.Now()
	for time.Now().Before(deadline) {
		now := fmt.Sprintf("%d|%s", len(w.lmStudio().allCalls()), w.run.output())
		if now != seen {
			seen, since = now, time.Now()
		} else if time.Since(since) >= idleFor {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("tgscout never settled in %s; output: %s", waitLimit, w.run.output())
}
