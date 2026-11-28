package e2e_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cucumber/godog"
)

// peerStoreSteps: the chats and the owner tgscout keeps in its database.
func peerStoreSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^tgscout found every chat of the config and the owner at an earlier start$`, func() error {
		w.request("kids-bike", bikeFind)
		w.matching.chats = []string{market, swap}
		if err := w.runToIdle(); err != nil {
			return err
		}
		calls, err := w.tg.calls(w.dir)
		if err != nil {
			return err
		}
		if len(resolveCalls(calls)) == 0 {
			return errors.New("the earlier start looked up nothing by name")
		}
		return w.forgetCalls()
	})
	sc.Step(`^the owner starts tgscout again with the same config$`, w.runToIdle)
	sc.Step(`^tgscout asks Telegram for no chat and no owner by name$`, func() error {
		calls, err := w.tg.calls(w.dir)
		if err != nil {
			return err
		}
		if resolved := resolveCalls(calls); len(resolved) > 0 {
			return fmt.Errorf("the second start looked up by name: %v", resolved)
		}
		return nil
	})
	sc.Step(`^it says that every watched chat came from the store$`, func() error {
		return w.stdoutHas("[ok] watching 2 chats · 2 from the store\n")
	})
}

// resolveCalls lists the chats and owners the fake was asked for by name.
func resolveCalls(calls []fakeCall) []string {
	var resolved []string
	for _, call := range calls {
		if call.Kind == "resolve" {
			resolved = append(resolved, call.Chat)
		}
	}
	return resolved
}

// forgetCalls removes the calls file, so the next start writes its own.
func (w *world) forgetCalls() error {
	err := os.Remove(filepath.Join(w.dir, callsFileName))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("calls file: %w", err)
	}
	return nil
}
