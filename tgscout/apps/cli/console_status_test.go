package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func TestPrintMatchShowsTheLatencyOfALiveMessage(t *testing.T) {
	c, out, _ := newTestConsole(true)
	message := madeUpMessage(t, "@made_up_market", 4211, "Selling a bike")
	message.Live = true
	c.now = func() time.Time { return message.SentAt.Add(3*time.Second + 420*time.Millisecond) }

	require.NoError(t, c.PrintMatch(domain.Match{Message: message, Request: domain.Request{ID: "kids-bike"}}, "Match: kids-bike"))

	assert.Equal(t, "[ok] match sent  14:02  kids-bike · latency 3.4s\n  Match: kids-bike\n", out.String())
}

func TestPrintMatchShowsNoLatencyForAHistoryOrEditedMessage(t *testing.T) {
	c, out, _ := newTestConsole(true)
	history := madeUpMessage(t, "@made_up_market", 4211, "Selling a bike")
	edited := history
	edited.Edited = true // an edit is never live

	require.NoError(t, c.PrintMatch(domain.Match{Message: history, Request: domain.Request{ID: "kids-bike"}}, "Match"))
	require.NoError(t, c.PrintMatch(domain.Match{Message: edited, Request: domain.Request{ID: "kids-bike"}}, "Match"))

	assert.NotContains(t, out.String(), "latency")
}

func TestStatusBlockWithinTheTarget(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.status(domain.Status{In: 312, Judged: 1904, Matches: 2, Queue: 4, Took: 5 * time.Minute,
		Latency: 3820 * time.Millisecond, HasLatency: true, LlmCall: 1100 * time.Millisecond, HasLlmCall: true})

	assert.Equal(t,
		"[run] status · last 5m\n"+
			"  in 312 · judged 1,904 · matches 2 · queue 4\n"+
			"  p95 latency 3.8s [ok] ≤ 10s · p95 LLM call 1.1s\n",
		out.String())
}

func TestStatusBlockOverTheTarget(t *testing.T) {
	c, out, _ := newTestConsole(false)

	c.status(domain.Status{In: 1240, Judged: 2010, Matches: 1, Queue: 388, Took: 5 * time.Minute,
		Latency: 41200 * time.Millisecond, HasLatency: true, LlmCall: 2900 * time.Millisecond, HasLlmCall: true})

	assert.Contains(t, uncolored(out.String()), "  p95 latency 41.2s ✗ > 10s · p95 LLM call 2.9s\n")
}

func TestStatusBlockWithoutSamplesShowsDashes(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.status(domain.Status{Queue: 0, Took: 10 * time.Second})

	assert.Equal(t,
		"[run] status · last 10s\n"+
			"  in 0 · judged 0 · matches 0 · queue 0\n"+
			"  p95 latency — · p95 LLM call —\n",
		out.String())
}

func TestStatusSinceStart(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.statusTotal(domain.Status{In: 4980, Judged: 29410, Matches: 9, Took: 8*time.Hour + 33*time.Minute + 12*time.Second,
		Latency: 4600 * time.Millisecond, HasLatency: true, LlmCall: 1200 * time.Millisecond, HasLlmCall: true})

	assert.Equal(t,
		"[ok] status since start (8h33m)\n"+
			"  in 4,980 · judged 29,410 · matches 9 · queue 0\n"+
			"  p95 latency 4.6s [ok] ≤ 10s · p95 LLM call 1.2s\n",
		out.String())
}

func TestCatchUpSummaryWithinItsTarget(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.catchUpDone(collect.Summary{Chats: 1, Took: 12*time.Minute + 29*time.Second})

	assert.Contains(t, out.String(), "[ok] catch-up done in 12m29s · [ok] ≤ 15m\n")
}

func TestCatchUpSummaryOverItsTarget(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.catchUpDone(collect.Summary{Chats: 100, Took: 3*time.Hour + 28*time.Minute})

	assert.Contains(t, out.String(),
		"[ok] catch-up done in 3h28m · [warn] > 15m — the model answers too slowly for this many messages\n")
}

func TestNonLocalLlmWarning(t *testing.T) {
	c, _, errOut := newTestConsole(true)

	c.nonLocalLlm("https://llm.example.net/v1")

	assert.Equal(t,
		"[warn] lm_studio.url is not on this machine: https://llm.example.net/v1\n"+
			"  chat texts will leave this machine; use localhost to keep them here\n",
		errOut.String())
}

// statusRun drives the status loop with one tick per queue length.
func statusRun(t *testing.T, queues ...int) string {
	t.Helper()
	c, out, errOut := newTestConsole(true)
	ticks := make(chan time.Time)
	at := 0
	cut := func(time.Time) domain.Status {
		status := domain.Status{Queue: queues[at], Took: 10 * time.Second}
		at++
		return status
	}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.runStatus(ctx, ticks, cut)
	}()
	for range queues {
		ticks <- time.Now()
	}
	stop()
	<-done
	return out.String() + errOut.String()
}

func TestStatusLoopPrintsALinePerTick(t *testing.T) {
	text := statusRun(t, 1, 1)

	assert.Equal(t, 2, countOf(text, "[run] status · last 10s"))
	assert.NotContains(t, text, "behind")
}

func TestStatusLoopWarnsAfterThreeGrowingLines(t *testing.T) {
	text := statusRun(t, 97, 214, 388)

	assert.Contains(t, text,
		"[warn] the model is behind: the queue grew 3 status lines in a row (97 → 214 → 388)\n"+
			"  use a faster model or more workers\n")
}

func TestStatusLoopGivesNoWarningWhenTheQueueShrinksOnce(t *testing.T) {
	text := statusRun(t, 10, 20, 15, 30)

	assert.NotContains(t, text, "behind")
}

func TestStatusLoopWarnsAgainOnlyAfterThreeMoreGrowingLines(t *testing.T) {
	assert.Equal(t, 1, countOf(statusRun(t, 1, 2, 3, 4, 5), "behind"))
	assert.Equal(t, 2, countOf(statusRun(t, 1, 2, 3, 4, 5, 6), "behind"))
}

func countOf(text, part string) int { return strings.Count(text, part) }
