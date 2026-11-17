package e2e_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cucumber/godog"
)

// The made-up stranger who writes to tgscout's account.
const stranger = "@made_up_stranger"

// sendFailedLine is the start of the warning of a failed send, up to its pause.
const sendFailedLine = "[warn] send failed: kids-bike on @made_up_market/4211 · cannot reach Telegram · retry in "

func ownerSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a request bound to a watched chat and the LLM answers yes$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
	})
	sc.Step(`^two requests of one chat and the LLM answers yes to both$`, func() {
		w.request("kids-bike", bikeFind)
		w.request("desk-lamp", lampFind)
		w.lmStudio().answers[bikeFind], w.lmStudio().answers[lampFind] = "yes", "yes"
	})
	sc.Step(`^every send of a match fails$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.SendFails = true
		w.matching.until = func() bool { return strings.Count(w.run.stderr.String(), sendFailedLine) >= 2 }
	})
	sc.Step(`^someone other than the owner writes to tgscout's account$`, func() {
		w.request("kids-bike", bikeFind)
		w.lmStudio().answers[bikeFind] = "yes"
		w.tg.fake.Updates = []fakeUpdate{{Chat: stranger, ID: 1, Text: bikeText, Private: true}}
	})
	sc.Step(`^(?:a new matching message appears in that chat|the matcher handles the message)$`, func() error {
		w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
		return w.runToIdle()
	})
	sc.Step(`^tgscout runs$`, w.runToIdle)
	ownerThenSteps(sc, w)
}

func ownerThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the owner gets one private message with the request, the chat and a link to the message$`, func() error {
		return w.ownerGot("Match: kids-bike — " + bikeFind + "\nChat: " + marketTitle +
			"\nhttps://t.me/made_up_market/4211\n" + bikeText)
	})
	sc.Step(`^the console shows the same match$`, func() error {
		return w.allStdoutHas("  kids-bike · latency ", "s\n  Match: kids-bike — "+bikeFind+"\n  Chat: "+marketTitle+
			"\n  https://t.me/made_up_market/4211\n  "+bikeText+"\n")
	})
	sc.Step(`^the owner gets two private messages, one per request$`, func() error {
		return w.ownerGot(
			"Match: kids-bike — "+bikeFind+"\nChat: "+marketTitle+"\nhttps://t.me/made_up_market/4211\n"+bikeText,
			"Match: desk-lamp — "+lampFind+"\nChat: "+marketTitle+"\nhttps://t.me/made_up_market/4211\n"+bikeText)
	})
	sc.Step(`^the message stays unprocessed and the console shows a warning line for each failed try$`, func() error {
		return errors.Join(w.processed(false), w.cursorAt(0), w.ownerGot(),
			w.allStderrHas(sendFailedLine+"1s · the message stays unprocessed\n",
				sendFailedLine+"2s · the message stays unprocessed\n"))
	})
	sc.Step(`^tgscout does not answer and writes nothing to that person$`, func() error {
		return errors.Join(w.ownerGot(), w.askedTimes(bikeFind, 0))
	})
	sc.Step(`^tgscout stops with an error that it cannot write to the owner and a hint to write «hi» to its account once$`,
		func() error {
			return w.stderrHas("[fail] cannot write to the owner",
				"write «hi» to tgscout's account once and add it to contacts (README → Setup → First contact)")
		})
}

// fakeSend is one line of the fake Telegram's sent file.
type fakeSend struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

// ownerGot checks the fake Telegram sent exactly these texts, in any order,
// all of them to the owner and to nobody else.
func (w *world) ownerGot(want ...string) error {
	got, err := w.sentTexts()
	if err != nil {
		return err
	}
	if len(got) != len(want) {
		return fmt.Errorf("the owner got %d messages, want %d: %q", len(got), len(want), got)
	}
	for _, text := range want {
		if !slices.Contains(got, text) {
			return fmt.Errorf("the owner never got %q; got %q", text, got)
		}
	}
	return nil
}

// sentTexts reads the fake Telegram's sent file; a send to anyone but the
// owner is an error.
func (w *world) sentTexts() ([]string, error) {
	data, err := os.ReadFile(filepath.Join(w.dir, sentFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sent file: %w", err)
	}
	var got []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var send fakeSend
		if err := json.Unmarshal([]byte(line), &send); err != nil {
			return nil, fmt.Errorf("sent file: %w", err)
		}
		if send.To != "@made_up_owner" {
			return nil, fmt.Errorf("tgscout wrote to %s", send.To)
		}
		got = append(got, send.Text)
	}
	return got, nil
}
