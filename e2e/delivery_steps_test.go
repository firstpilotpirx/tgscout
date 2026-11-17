package e2e_test

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/cucumber/godog"
)

// The console lines of the delivery scenarios.
const (
	llmDownLine = "[warn] LM Studio is not answering · retry in 1s, 2s, 4s … up to "
	llmBackLine = "[ok] LM Studio is back · "
	bikeMatch   = "Match: kids-bike — " + bikeFind + "\nChat: " + marketTitle + "\nhttps://t.me/made_up_market/4211\n" + bikeText
)

func deliverySteps(sc *godog.ScenarioContext, w *world) {
	deliveryGivenSteps(sc, w)
	deliveryWhenSteps(sc, w)
	deliveryCrashSteps(sc, w)
	deliveryThenSteps(sc, w)
}

func deliveryGivenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^LM Studio stops answering and answers yes once it is back$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.lmStudio().down = true
		w.matching.extra = "retry_max_pause: 1s\n"
	})
	sc.Step(`^a queue size of 1, one worker and LM Studio that does not answer$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().down = true
		w.matching.workers = 1
		w.matching.extra = "queue_size: 1\nretry_max_pause: 1s\n"
	})
	sc.Step(`^the LLM answers yes to a request on a watched chat$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
	})
	sc.Step(`^LM Studio answers after (\d+) seconds?$`, func(seconds int) {
		w.request("kids-bike", bikeFind)
		w.lmStudio().delay = time.Duration(seconds) * time.Second
	})
	sc.Step(`^a grace time of 1 second and LM Studio that answers after 5 seconds$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().delay = 5 * time.Second
		w.matching.extra = "stop_grace: 1s\n"
	})
}

func deliveryWhenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a new message appears in a watched chat and LM Studio comes back$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		return w.runUntil(func() bool { return strings.Contains(w.run.stderr.String(), llmDownLine) },
			func() { w.lmStudio().setDown(false) },
			func() bool { return w.processedCount() == 1 })
	})
	sc.Step(`^three new messages appear and LM Studio comes back after the queue is full$`, func() error {
		w.tg.fake.Updates = threeMessages()
		return w.runUntil(func() bool { return strings.Contains(w.run.stderr.String(), "queue is full") },
			func() { w.lmStudio().setDown(false) },
			func() bool { return w.processedCount() == 3 })
	})
	sc.Step(`^the connection drops, a message is posted during the drop and the connection is back$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{
			{ConnectionLost: true, AfterMs: 300},
			{Chat: market, ID: firstID, Text: bikeText, Lost: true, AfterMs: 100},
			{ConnectionBack: true, AfterMs: 100},
		}
		w.matching.until = func() bool { return strings.Contains(w.run.stdout.String(), "gap read done") }
		return w.runToIdle()
	})
	sc.Step(`^the owner presses Ctrl\+C while a message is judged$`, func() error {
		if err := w.startJudging(); err != nil {
			return err
		}
		return w.run.send(syscall.SIGINT)
	})
	sc.Step(`^the owner presses Ctrl\+C while the message waits for its next try$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		if err := w.startMatching(); err != nil {
			return err
		}
		if err := w.eventually("the not-answering line", func() bool {
			return strings.Contains(w.run.stderr.String(), llmDownLine)
		}); err != nil {
			return err
		}
		return w.run.send(syscall.SIGINT)
	})
	sc.Step(`^the owner presses Ctrl\+C twice while a message is judged$`, func() error {
		if err := w.startJudging(); err != nil {
			return err
		}
		if err := w.run.send(syscall.SIGINT); err != nil {
			return err
		}
		if err := w.eventually("the stopping line", func() bool {
			return strings.Contains(w.run.stdout.String(), "-> stopping · ")
		}); err != nil {
			return err
		}
		return w.run.send(syscall.SIGINT)
	})
}

func deliveryCrashSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the LLM answers yes and the confirmation of the send never comes$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.SendHangMs = 60_000
	})
	sc.Step(`^the process is killed after the send and the owner starts tgscout again$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		if err := w.startMatching(); err != nil {
			return err
		}
		if err := w.eventually("the send", func() bool { got, _ := w.sentTexts(); return len(got) == 1 }); err != nil {
			return err
		}
		if err := w.kill(); err != nil {
			return err
		}
		w.tg.fake.SendHangMs, w.tg.fake.Updates = 0, nil
		w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 1}}
		return w.runToIdle()
	})
	sc.Step(`^one worker and LM Studio that never answers in time$`, func() {
		w.request("kids-bike", bikeFind)
		w.matching.workers = 1
		w.lmStudio().delay = time.Minute
	})
	sc.Step(`^three messages arrive, the process is killed and the owner starts tgscout again$`, func() error {
		w.tg.fake.Updates = threeMessages()
		if err := w.startJudging(); err != nil {
			return err
		}
		if err := w.kill(); err != nil {
			return err
		}
		w.matching.before = len(w.lmStudio().allCalls())
		w.lmStudio().setDelay(0)
		w.tg.fake.Updates = nil
		for _, each := range threeMessages() {
			w.tg.fake.History = append(w.tg.fake.History, fakeHistory{Chat: market, ID: each.ID, Text: each.Text, AgeMinutes: 1})
		}
		return w.runToIdle()
	})
	sc.Step(`^every message read so far is processed after a match$`, func() error {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		if err := w.runToIdle(); err != nil {
			return err
		}
		return w.processed(true)
	})
	sc.Step(`^the owner stops and starts tgscout$`, func() error {
		w.tg.fake.Updates = nil
		w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 1}}
		return w.runToIdle()
	})
}

func deliveryThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the owner gets the match, none lost$`, func() error { return errors.Join(w.ownerGot(bikeMatch), w.processed(true)) })
	sc.Step(`^the console shows one line that LM Studio is not answering and one that it is back$`, func() error {
		return errors.Join(w.lineCount(true, llmDownLine, 1), w.lineCount(false, llmBackLine, 1))
	})
	sc.Step(`^the console shows that the queue is full$`, func() error {
		return w.allStderrHas("[warn] queue is full (1) · reading new messages paused\n")
	})
	sc.Step(`^every one of the three messages is judged and marked processed$`, func() error {
		if got := w.processedCount(); got != 3 {
			return fmt.Errorf("%d messages processed, want 3", got)
		}
		return nil
	})
	sc.Step(`^the message posted during the drop is judged and the owner gets its match$`, func() error {
		return errors.Join(w.askedTimes(bikeFind, 1), w.ownerGot(bikeMatch), w.processed(true))
	})
	sc.Step(`^the console shows that the connection was lost, that it is back and the gap read$`, func() error {
		return errors.Join(w.allStderrHas("[warn] Telegram connection lost · reconnecting\n"),
			w.allStdoutHas("[ok] Telegram connection back · reading 1 chat from their cursors\n",
				"[ok] gap read done · 1 message to judge\n"))
	})
	sc.Step(`^the console shows the stopping line with 1 message in work$`, func() error {
		return errors.Join(w.run.exited(),
			w.stdoutHas("-> stopping · finishing 1 message in work · up to 10s · Ctrl+C again to stop now\n"))
	})
	sc.Step(`^tgscout exits with code 0 and the message in work is marked processed$`, func() error {
		return errors.Join(w.exitCode(func(c int) bool { return c == 0 }), w.processed(true))
	})
	sc.Step(`^tgscout exits with code 0 within (\d+) seconds?$`, func(seconds int) error {
		took, err := w.run.exitedWithin()
		if err != nil {
			return err
		}
		if limit := time.Duration(seconds) * time.Second; took > limit {
			return fmt.Errorf("tgscout took %s to exit, want under %s", took, limit)
		}
		return w.exitCode(func(c int) bool { return c == 0 })
	})
	sc.Step(`^tgscout ends at once with exit code 130$`, func() error {
		took, err := w.run.exitedWithin()
		if err != nil {
			return err
		}
		if took > time.Second {
			return fmt.Errorf("tgscout took %s to end after the second Ctrl+C", took)
		}
		return w.exitCode(func(c int) bool { return c == 130 })
	})
	sc.Step(`^the owner gets that match again, and no match is lost$`, func() error {
		return errors.Join(w.ownerGot(bikeMatch, bikeMatch), w.processed(true))
	})
	sc.Step(`^every one of those messages is judged after the restart$`, func() error {
		var got []string
		for _, call := range w.lmStudio().allCalls()[w.matching.before:] {
			got = append(got, call.Message)
		}
		for _, each := range threeMessages() {
			if !strings.Contains(strings.Join(got, "\n"), each.Text) {
				return fmt.Errorf("%q was not judged after the restart; asked about %q", each.Text, got)
			}
		}
		return nil
	})
	sc.Step(`^no match is sent again and the LLM is not asked again$`, func() error {
		return errors.Join(w.ownerGot(bikeMatch), w.askedTimes(bikeFind, 1))
	})
}

// threeMessages are three made-up messages of the market, played one after another.
func threeMessages() []fakeUpdate {
	return []fakeUpdate{
		{Chat: market, ID: firstID, Text: "Kids' bike 1"},
		{Chat: market, ID: firstID + 1, Text: "Kids' bike 2"},
		{Chat: market, ID: firstID + 2, Text: "Kids' bike 3"},
	}
}

// runUntil starts tgscout, waits for ready, acts, waits for done, then stops it.
func (w *world) runUntil(ready func() bool, act func(), done func() bool) error {
	if err := w.startMatching(); err != nil {
		return err
	}
	if err := w.eventually("the moment to act", ready); err != nil {
		return err
	}
	act()
	if err := w.eventually("the end of the work", done); err != nil {
		return err
	}
	return w.run.signal(syscall.SIGINT)
}

// startJudging starts tgscout with one message (unless the scenario has
// updates) and waits until LM Studio is asked about it.
func (w *world) startJudging() error {
	if len(w.tg.fake.Updates) == 0 {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
	}
	calls := len(w.lmStudio().allCalls())
	if err := w.startMatching(); err != nil {
		return err
	}
	return w.eventually("a call to LM Studio", func() bool { return len(w.lmStudio().allCalls()) > calls })
}

// kill ends tgscout with SIGKILL, as a crash.
func (w *world) kill() error {
	if err := w.run.send(syscall.SIGKILL); err != nil {
		return err
	}
	return w.run.exited()
}

func (w *world) processedCount() int {
	count, err := w.query(`SELECT count(*) FROM processed_message`)
	if err != nil {
		return -1
	}
	return count
}

// lineCount checks how many lines of every run start with prefix.
func (w *world) lineCount(stderr bool, prefix string, want int) error {
	got := 0
	for _, line := range strings.Split(w.allOut(stderr), "\n") {
		if strings.HasPrefix(line, prefix) {
			got++
		}
	}
	if got != want {
		return fmt.Errorf("%d lines start with %q, want %d", got, prefix, want)
	}
	return nil
}
