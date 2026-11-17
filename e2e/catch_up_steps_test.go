package e2e_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cucumber/godog"
)

// The made-up chats of the catch-up scenarios beside the market.
const (
	swap   = "@made_up_swap"
	street = "@made_up_street"
	hour   = 60
	day    = 24 * hour
)

func catchUpSteps(sc *godog.ScenarioContext, w *world) {
	catchUpGivenSteps(sc, w)
	sc.Step(`^the owner starts tgscout and it catches up$`, w.runToIdle)
	sc.Step(`^a new message appears in that chat while it is read$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		return w.runToIdle()
	})
	catchUpThenSteps(sc, w)
}

func catchUpGivenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a matching message posted while tgscout was off$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 30}}
	})
	sc.Step(`^a watched chat with no cursor and 4 days of history$`, func() {
		w.request("kids-bike", bikeFind)
		w.tg.fake.History = []fakeHistory{
			{Chat: market, ID: 1, Text: "Message 1", AgeMinutes: 4 * day},
			{Chat: market, ID: 2, Text: "Message 2", AgeMinutes: 2 * day},
			{Chat: market, ID: 3, Text: "Message 3", AgeMinutes: hour},
		}
	})
	sc.Step(`^a chat whose cursor is at message 100 and 20 newer messages$`, func() error {
		w.request("kids-bike", bikeFind)
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: 100, Text: "Message 100"}}
		if err := w.runToIdle(); err != nil {
			return err
		}
		w.tg.fake.Updates = nil
		w.tg.fake.History = numbered(market, 90, 120)
		return nil
	})
	sc.Step(`^catch-up is reading a chat$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.HistoryDelayMs = 600 // the live message arrives while the history is read
		w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 1}}
	})
	sc.Step(`^Telegram asks to wait 1 second on the first history read$`, func() {
		w.request("kids-bike", bikeFind)
		w.tg.fake.HistoryFlood = 1
		w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 30}}
	})
	sc.Step(`^two watched chats, one with a cursor and one without$`, func() error {
		w.request("kids-bike", bikeFind)
		w.matching.chats = []string{market, swap}
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: 10, Text: "Message 10"}}
		if err := w.runToIdle(); err != nil {
			return err
		}
		w.tg.fake.Updates = nil
		w.tg.fake.History = append(numbered(market, 9, 12), fakeHistory{Chat: swap, ID: 5, Text: "Swap 5", AgeMinutes: hour})
		return nil
	})
	sc.Step(`^three watched chats with history$`, func() {
		w.request("kids-bike", bikeFind)
		w.matching.chats = []string{market, swap, street}
		w.tg.fake.History = []fakeHistory{
			{Chat: market, ID: 1, Text: "Market 1", AgeMinutes: hour},
			{Chat: swap, ID: 1, Text: "Swap 1", AgeMinutes: hour},
			{Chat: street, ID: 1, Text: "Street 1", AgeMinutes: hour},
		}
	})
	sc.Step(`^a watched chat with history and LM Studio that does not answer$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().down = true
		w.tg.fake.History = []fakeHistory{{Chat: market, ID: firstID, Text: bikeText, AgeMinutes: 30}}
	})
}

func catchUpThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the owner gets the history match as a private message$`, func() error {
		return w.ownerGot("Match: kids-bike — " + bikeFind + "\nChat: " + marketTitle +
			"\nhttps://t.me/made_up_market/4211\n" + bikeText)
	})
	sc.Step(`^tgscout judges the messages of the last 3 days only$`, func() error {
		return w.judged("Message 2", "Message 3")
	})
	sc.Step(`^only the 20 newer messages are judged$`, func() error {
		want := []string{"Message 100"} // judged live by the first start
		for id := 101; id <= 120; id++ {
			want = append(want, fmt.Sprintf("Message %d", id))
		}
		return w.judged(want...)
	})
	sc.Step(`^that message is judged once and the owner gets one match for it$`, func() error {
		return errors.Join(w.askedTimes(bikeFind, 1), w.ownerGot("Match: kids-bike — "+bikeFind+"\nChat: "+marketTitle+
			"\nhttps://t.me/made_up_market/4211\n"+bikeText))
	})
	sc.Step(`^the console shows the history wait$`, func() error {
		return w.allStderrHas("[warn] Telegram asks to wait 1s (FLOOD_WAIT), then goes on")
	})
	sc.Step(`^the chat's history is read and judged after it$`, func() error {
		return errors.Join(w.askedTimes(bikeFind, 1), w.allStdoutHas("[ok] catch-up: "+marketTitle+" · 1 message · first start"))
	})
	sc.Step(`^the console shows a start line and one line per chat, in config order$`, func() error {
		out := w.run.stdout.String()
		start := strings.Index(out, "-> catch-up started · 2 chats · live listening is on\n")
		first := strings.Index(out, "[ok] catch-up: "+marketTitle+" · 2 messages from the cursor\n")
		second := strings.Index(out, "[ok] catch-up: "+swap+" · 1 message")
		if start < 0 || first < start || second < first {
			return fmt.Errorf("want the start line, then the market, then the swap; stdout:\n%s", out)
		}
		return nil
	})
	sc.Step(`^the line of the chat without a cursor says it is a first start$`, func() error {
		return w.stdoutHas("[ok] catch-up: " + swap + " · 1 message · first start, last 3 days\n")
	})
	sc.Step(`^the console shows the summary with chats, messages, matches and the time$`, func() error {
		return w.stdoutHas("[ok] catch-up done in ", "  [ok] chats      3\n",
			"  [ok] messages   3 read · 0 skipped · 3 judged\n",
			"  [run] matches    0  (sent to you in Telegram)\n", "  [ok] errors     0\n")
	})
	sc.Step(`^the summary counts the failed messages as errors$`, func() error {
		return w.stdoutHas("  [fail] errors     1  (not marked processed; judged again at next start)\n")
	})
	sc.Step(`^those messages are not marked processed$`, func() error { return w.processed(false) })
}

// numbered gives the history messages from..to of a chat, one minute apart,
// the newest a minute old.
func numbered(chat string, from, to int) []fakeHistory {
	var history []fakeHistory
	for id := from; id <= to; id++ {
		history = append(history, fakeHistory{Chat: chat, ID: id, Text: fmt.Sprintf("Message %d", id), AgeMinutes: to - id + 1})
	}
	return history
}

// judged checks the LLM was asked about exactly these message texts, in any order.
func (w *world) judged(want ...string) error {
	var got []string
	for _, call := range w.lmStudio().callsAbout(bikeFind) {
		got = append(got, call.Message)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("the LLM was asked about %q, want %q", got, want)
	}
	return nil
}
