package e2e_test

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/cucumber/godog"
)

// The made-up requests and messages of the matching scenarios.
const (
	market       = "@made_up_market"
	bikeFind     = "a kids' bike, 16 inch, under 100 €"
	lampFind     = "a desk lamp with a clamp"
	bikeText     = "Selling a 16-inch kids' bike, 80 €"
	firstID      = 4211
	unclearReply = "It might be, the size is not stated"
	settleGap    = 300 // ms between two updates, so the first is processed first
	idleFor      = 400 * time.Millisecond
)

// matchRequest is one request of the scenario's config.
type matchRequest struct{ id, find string }

// matchingWorld is the matching side of a scenario: the requests of its
// config and the fake LM Studio.
type matchingWorld struct {
	requests []matchRequest
	lm       *fakeLmStudio
	runs     []*process // every start of the scenario, in order
	chats    []string   // the watched chats of every request; the market when empty
	workers  int        // 2 when zero
	extra    string     // more root lines of the config: queue_size, stop_grace, retry_max_pause
	until    func() bool
	before   int // LM Studio calls before a restart
}

func matchingSteps(sc *godog.ScenarioContext, w *world) {
	matchingGivenSteps(sc, w)
	matchingWhenSteps(sc, w)
	matchingThenSteps(sc, w)
	consoleThenSteps(sc, w)
	decisionThenSteps(sc, w)
	catchUpSteps(sc, w)
}

func matchingGivenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^two requests bound to the same watched chat$`, func() {
		w.request("kids-bike", bikeFind)
		w.request("desk-lamp", lampFind)
	})
	sc.Step(`^a request "([^"]*)" bound to a watched chat and the model example-model$`, func(find string) {
		w.request("desk-lamp", find)
	})
	sc.Step(`^a request "([^"]*)" bound to a watched chat, and the LLM answers (yes|no)$`, func(find, answer string) {
		w.request("kids-bike", find)
		w.lmStudio().answers[find] = answer
	})
	sc.Step(`^the LLM answers (yes|no)$`, func(answer string) {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = answer
	})
	sc.Step(`^the LLM answers neither yes nor no, twice$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = unclearReply
	})
	sc.Step(`^LM Studio does not answer$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().down = true
	})
	sc.Step(`^a message that is already marked processed$`, func() error {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		return w.runToIdle()
	})
	sc.Step(`^a processed message with some text$`, func() {
		w.request("kids-bike", bikeFind)
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
	})
	sc.Step(`^a processed message with a «yes» for one request and a «no» for another$`, func() {
		w.request("kids-bike", bikeFind)
		w.request("desk-lamp", lampFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText + ", and a desk lamp"}}
	})
}

func matchingWhenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a new message appears in (?:that|a watched) chat$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		return w.runToIdle()
	})
	sc.Step(`^a new message "([^"]*)" appears in that chat$`, func(text string) error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: text}}
		return w.runToIdle()
	})
	sc.Step(`^the same message arrives again$`, func() error {
		// once more in a run, and once in a second start over the same database
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText},
			{Chat: market, ID: firstID, Text: bikeText, AfterMs: settleGap}}
		return w.runToIdle()
	})
	sc.Step(`^another message with the same text appears in the same chat$`, func() error {
		w.tg.fake.Updates = append(w.tg.fake.Updates, fakeUpdate{Chat: market, ID: firstID + 1, Text: bikeText, AfterMs: settleGap})
		return w.runToIdle()
	})
	sc.Step(`^the message is edited$`, func() error {
		first := w.tg.fake.Updates[0]
		w.tg.fake.Updates = append(w.tg.fake.Updates,
			fakeUpdate{Chat: market, ID: first.ID, Text: first.Text + " (edited)", Edited: true, AfterMs: settleGap})
		return w.runToIdle()
	})
}

func matchingThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the LLM is asked once per request$`, func() error {
		return errors.Join(w.askedTimes(bikeFind, 1), w.askedTimes(lampFind, 1))
	})
	sc.Step(`^the LLM is asked twice about that request$`, func() error { return w.askedTimes(bikeFind, 2) })
	sc.Step(`^the LLM is not asked again and no second match line appears$`, func() error {
		return errors.Join(w.askedTimes(bikeFind, 1), w.matchLines("kids-bike", 1))
	})
	sc.Step(`^the LLM is asked about the new message$`, func() error {
		calls := w.lmStudio().callsAbout(bikeFind)
		if len(calls) != 2 || calls[1].Message != bikeText {
			return fmt.Errorf("want the same text asked twice, got %d calls: %+v", len(calls), calls)
		}
		return nil
	})
	sc.Step(`^the LLM is asked only about the request with «no»$`, func() error {
		return errors.Join(w.askedTimes(bikeFind, 1), w.askedTimes(lampFind, 2))
	})
	sc.Step(`^no second match line appears for the other$`, func() error { return w.matchLines("kids-bike", 1) })
	sc.Step(`^the LLM is called with the model example-model$`, func() error {
		calls := w.lmStudio().allCalls()
		if len(calls) == 0 {
			return errors.New("the LLM was not called")
		}
		for _, each := range calls {
			if each.Model != "example-model" {
				return fmt.Errorf("called with the model %q", each.Model)
			}
		}
		return nil
	})
	sc.Step(`^the call holds the request text and the message text apart from the fixed matching rules$`, w.textsApart)
}

func consoleThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the console shows a warning line with the raw answer$`, func() error {
		return w.allStderrHas("[warn] unclear LLM answer for kids-bike on @made_up_market/4211, saved as no",
			`answer: "`+unclearReply+`"`)
	})
	sc.Step(`^the console shows a match line with the request, the chat and the link to the message when the chat has one$`, func() error {
		return w.allStdoutHas("[ok] match sent  ", "  kids-bike · latency ", "s\n  Match: kids-bike — "+bikeFind+"\n  Chat: "+marketTitle+
			"\n  https://t.me/made_up_market/4211\n  "+bikeText+"\n")
	})
	sc.Step(`^the console shows no match line$`, func() error { return w.matchLines("", 0) })
	sc.Step(`^the console shows a line that LM Studio is not answering$`, func() error {
		return w.allStderrHas("[warn] LM Studio is not answering · retry in 1s, 2s, 4s … up to 60s · messages wait in the queue\n")
	})
}

func decisionThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^two decisions are saved, one per request$`, func() error {
		return errors.Join(w.decided("kids-bike", false), w.decided("desk-lamp", false))
	})
	sc.Step(`^the decision is saved as no match$`, func() error { return w.decided("kids-bike", false) })
	sc.Step(`^the decision is saved as a match, the message is marked processed and the cursor moves to it$`, func() error {
		return errors.Join(w.decided("kids-bike", true), w.processed(true), w.cursorAt(firstID))
	})
	sc.Step(`^the decision is saved as no match and the message is marked processed$`, func() error {
		return errors.Join(w.decided("kids-bike", false), w.processed(true))
	})
	sc.Step(`^the message is not marked processed and the cursor does not move past it$`, func() error {
		return errors.Join(w.processed(false), w.cursorAt(0))
	})
}

func (w *world) lmStudio() *fakeLmStudio {
	if w.matching.lm == nil {
		w.matching.lm = newFakeLmStudio()
	}
	return w.matching.lm
}

func (w *world) request(id, find string) {
	w.matching.requests = append(w.matching.requests, matchRequest{id: id, find: find})
}

// matchingConfig is the made-up config of the scenario's requests, all on one chat.
func (w *world) matchingConfig() string {
	var text strings.Builder
	workers := w.matching.workers
	if workers == 0 {
		workers = 2
	}
	fmt.Fprintf(&text, "owner: '@made_up_owner'\nworkers: %d\n%slm_studio:\n  url: %s\n  model: example-model\n  timeout: 5s\nrequests:\n",
		workers, w.matching.extra, w.lmStudio().url())
	chats := w.matching.chats
	if len(chats) == 0 {
		chats = []string{market}
	}
	for _, each := range w.matching.requests {
		fmt.Fprintf(&text, "  - id: %s\n    find: %q\n    chats: ['%s']\n", each.id, each.find, strings.Join(chats, "', '"))
	}
	return text.String()
}

// startMatching starts tgscout over the scenario's config and database and
// leaves it running.
func (w *world) startMatching() error {
	if err := w.writeConfig(w.matchingConfig()); err != nil {
		return err
	}
	if err := w.start(); err != nil {
		return err
	}
	w.matching.runs = append(w.matching.runs, w.run)
	return nil
}

// runToIdle starts tgscout over the scenario's database, waits until every
// update was played and judged (and the scenario's until holds), then stops it.
func (w *world) runToIdle() error {
	if err := w.startMatching(); err != nil {
		return err
	}
	last := w.tg.fake.HistoryDelayMs + w.tg.fake.HistoryFlood*1000 // the history read before it settles
	updates := 0
	for _, each := range w.tg.fake.Updates {
		updates += each.AfterMs // each update waits its after_ms after the one before
	}
	last = max(last, updates)
	time.Sleep(time.Duration(last) * time.Millisecond)
	if w.matching.until != nil {
		if err := w.eventually("the scenario's condition", w.matching.until); err != nil {
			return err
		}
	}
	if err := w.waitIdle(); err != nil {
		return err
	}
	return w.run.signal(syscall.SIGINT)
}

func (w *world) askedTimes(find string, want int) error {
	if got := len(w.lmStudio().callsAbout(find)); got != want {
		return fmt.Errorf("the LLM was asked %d times about %q, want %d", got, find, want)
	}
	return nil
}

func (w *world) textsApart() error {
	for _, each := range w.lmStudio().allCalls() {
		if strings.Contains(each.System, each.Request) || strings.Contains(each.System, each.Message) {
			return errors.New("the fixed rules hold the request or the message text")
		}
		if each.Request != lampFind || each.Message != "Desk lamp, clamp mount, 10 €" {
			return fmt.Errorf("the call holds %q", each.Original)
		}
	}
	return nil
}

func (w *world) query(text string, args ...any) (int, error) {
	db, err := w.openDB()
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	var value sql.NullInt64
	if err := db.QueryRow(text, args...).Scan(&value); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("read the database: %w", err)
	}
	return int(value.Int64), nil
}

func (w *world) decided(request string, matched bool) error {
	count, err := w.query(`SELECT count(*) FROM decision WHERE message_id = ? AND request_id = ? AND matched = ?`,
		firstID, request, matched)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("no decision of %s with matched=%t on message %d", request, matched, firstID)
	}
	return nil
}

func (w *world) processed(want bool) error {
	count, err := w.query(`SELECT count(*) FROM processed_message WHERE message_id = ?`, firstID)
	if err != nil {
		return err
	}
	if (count == 1) != want {
		return fmt.Errorf("message %d processed: %t, want %t", firstID, count == 1, want)
	}
	return nil
}

func (w *world) cursorAt(want int) error {
	got, err := w.query(`SELECT max(last_message_id) FROM chat_cursor`)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("the cursor is at %d, want %d", got, want)
	}
	return nil
}
