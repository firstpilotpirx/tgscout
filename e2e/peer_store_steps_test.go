package e2e_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	goneChatSteps(sc, w)
}

// goneChatSteps: a stored chat Telegram no longer knows under its stored data.
func goneChatSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a stored chat that Telegram no longer knows under its stored data, though the account is still in it$`,
		func() error {
			w.request("kids-bike", bikeFind)
			w.lmStudio().answers[bikeFind] = "yes"
			if err := w.runToIdle(); err != nil {
				return err
			}
			w.tg.fake.GoneChats = []string{market}
			w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 30}}
			return nil
		})
	sc.Step(`^catch-up reads that chat$`, func() error {
		w.untilCatchUpDone()
		return w.runToIdle()
	})
	sc.Step(`^the console shows one line that tgscout looked the chat up again$`, func() error {
		return w.outLineCount("[warn] catch-up: "+marketTitle+" · Telegram no longer knows the stored chat · looked it up again", 1)
	})
	sc.Step(`^the chat's history is read and judged$`, func() error {
		return errors.Join(w.judged(bikeText), w.allStdoutHas("[ok] catch-up: "+marketTitle+" · 1 message"))
	})
}

// outLineCount checks how many lines of every run, stdout and stderr, are line.
func (w *world) outLineCount(line string, want int) error {
	got := 0
	for _, each := range strings.Split(w.allOut(false)+w.allOut(true), "\n") {
		if each == line {
			got++
		}
	}
	if got != want {
		return fmt.Errorf("%d lines are %q, want %d; stdout:\n%s\nstderr:\n%s", got, line, want, w.allOut(false), w.allOut(true))
	}
	return nil
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
