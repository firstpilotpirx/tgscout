package e2e_test

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/cucumber/godog"
)

const (
	remoteLlm      = "http://llm.example.invalid/v1"
	leaveWarning   = "chat texts will leave this machine"
	statusInterval = 10 * time.Second
)

var (
	latencyLine = regexp.MustCompile(`(?m)^\[ok\] match sent  \d\d:\d\d  kids-bike · latency \d+\.\ds$`)
	p95Line     = regexp.MustCompile(`(?m)^  p95 latency \d+\.\ds \[ok\] ≤ 10s · p95 LLM call \d+\.\ds$`)
	sinceStart  = regexp.MustCompile(`(?m)^\[ok\] status since start \(\d+s\)\n  in 1 · judged 1 · matches 1 · queue 0\n` +
		`  p95 latency \d+\.\ds \[ok\] ≤ 10s · p95 LLM call \d+\.\ds\n\[ok\] stopped\n`)
)

func qualitySteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the console match line shows the latency from the message to the send$`, func() error {
		return w.matches(latencyLine, "a match line with its latency")
	})
	sc.Step(`^the console match line of that message shows no latency$`, func() error {
		if err := w.matchLines("kids-bike", 1); err != nil {
			return err
		}
		if regexp.MustCompile(`(?m)^\[ok\] match sent .*latency`).MatchString(w.allOut(false)) {
			return fmt.Errorf("a latency on a catch-up match; stdout:\n%s", w.allOut(false))
		}
		return nil
	})
	sc.Step(`^tgscout runs with a status interval of 10 seconds and judges live messages$`, func() error {
		w.oneLiveMatch()
		w.matching.extra = "status_interval: 10s\n"
		return w.startMatching()
	})
	sc.Step(`^the status interval passes$`, func() error {
		return w.eventuallyWithin("a status line", statusInterval+5*time.Second, func() bool {
			return strings.Contains(w.run.stdout.String(), "[run] status · last 10s\n")
		})
	})
	sc.Step(`^the console shows messages in, messages judged, matches, queue length and the p95 latency with its target mark$`,
		func() error {
			return errors.Join(w.stdoutHas("[run] status · last 10s\n  in 1 · judged 1 · matches 1 · queue 0\n"),
				w.matches(p95Line, "the p95 line with its mark"))
		})
	sc.Step(`^tgscout ran and judged live messages$`, w.oneLiveMatch)
	sc.Step(`^the owner presses Ctrl\+C$`, w.runToIdle)
	sc.Step(`^before the stopped line the console shows the status since start with how long tgscout ran, the counts and the p95 latency with its target mark$`,
		func() error { return w.matches(sinceStart, "the status since start before the stopped line") })
	qualityLlmSteps(sc, w)
}

func qualityLlmSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a config whose LM Studio address points to another machine$`, func() error {
		return w.writeConfig(strings.Replace(validConfig, "lm_studio:\n", "lm_studio:\n  url: "+remoteLlm+"\n", 1))
	})
	sc.Step(`^a config whose LM Studio address is on this machine$`, func() error { return w.writeConfig(validConfig) })
	sc.Step(`^tgscout warns that chat texts will leave this machine and names the address$`, func() error {
		return w.stderrHas("[warn] lm_studio.url is not on this machine: "+remoteLlm, leaveWarning)
	})
	sc.Step(`^it keeps running$`, func() error {
		select {
		case <-w.run.done:
			return fmt.Errorf("tgscout exited with %d; output: %s", w.run.code, w.run.output())
		default:
		}
		if err := w.stdoutHas("running — press Ctrl+C to stop"); err != nil {
			return err
		}
		return w.run.signal(syscall.SIGINT)
	})
	sc.Step(`^tgscout prints no warning about chat texts leaving the machine$`, func() error {
		if strings.Contains(w.run.output(), leaveWarning) {
			return fmt.Errorf("a warning about a local LLM; output: %s", w.run.output())
		}
		return nil
	})
}

// oneLiveMatch is one request the LLM says yes to and one live message.
func (w *world) oneLiveMatch() {
	w.request("kids-bike", bikeFind)
	w.lmStudio().answers[bikeFind] = "yes"
	w.tg.fake.Updates = []fakeUpdate{{Chat: market, ID: firstID, Text: bikeText}}
}

// matches checks the stdout of every run against a pattern.
func (w *world) matches(pattern *regexp.Regexp, what string) error {
	if !pattern.MatchString(w.allOut(false)) {
		return fmt.Errorf("no %s; stdout:\n%s", what, w.allOut(false))
	}
	return nil
}

// eventuallyWithin waits until done holds, at most limit.
func (w *world) eventuallyWithin(what string, limit time.Duration, done func() bool) error {
	deadline := time.Now().Add(limit)
	for !done() {
		if time.Now().After(deadline) {
			return fmt.Errorf("%s never held in %s; output: %s", what, limit, w.run.output())
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
