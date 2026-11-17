package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// The targets of version 1 the console marks (docs/architecture/quality-attributes.html).
const (
	latencyTarget = 10 * time.Second
	catchUpTarget = 15 * time.Minute
	// behindLines is how many status lines in a row the queue must grow in
	// before the console says the model is behind.
	behindLines = 3
)

// status prints the status block of the last interval.
func (c *console) status(s domain.Status) {
	c.statusBlock(c.cyan.Sprint(c.marks.run)+" status · last "+short(s.Took), s)
}

// statusTotal prints the status since start, at the stop.
func (c *console) statusTotal(s domain.Status) {
	took := s.Took
	if took >= time.Hour {
		took = took.Truncate(time.Minute)
	}
	c.statusBlock(fmt.Sprintf("%s status since start (%s)", c.green.Sprint(c.marks.ok), short(took)), s)
}

func (c *console) statusBlock(head string, s domain.Status) {
	latency := "p95 latency —"
	if s.HasLatency {
		latency = fmt.Sprintf("p95 latency %s %s", tenths(s.Latency), c.latencyMark(s.Latency))
	}
	call := "p95 LLM call —"
	if s.HasLlmCall {
		call = "p95 LLM call " + tenths(s.LlmCall)
	}
	_ = c.block(c.out, head, // a lost line cannot be reported
		fmt.Sprintf("  in %s · judged %s · matches %s · queue %s",
			thousands(s.In), thousands(s.Judged), thousands(s.Matches), thousands(s.Queue)),
		"  "+latency+" · "+call)
}

func (c *console) latencyMark(latency time.Duration) string {
	if latency <= latencyTarget {
		return c.green.Sprint(c.marks.ok) + " ≤ " + short(latencyTarget)
	}
	return c.red.Sprint(c.marks.fail) + " > " + short(latencyTarget)
}

// catchUpMark marks the catch-up time against its target.
func (c *console) catchUpMark(took time.Duration) string {
	if took <= catchUpTarget {
		return c.green.Sprint(c.marks.ok) + " ≤ " + short(catchUpTarget)
	}
	return c.yellow.Sprint(c.marks.warn) + " > " + short(catchUpTarget) +
		" — the model answers too slowly for this many messages"
}

// behind says the queue grew in the last status lines.
func (c *console) behind(queues []int) {
	lengths := make([]string, len(queues))
	for i, each := range queues {
		lengths[i] = thousands(each)
	}
	_ = c.block(c.errOut, // a lost line cannot be reported
		fmt.Sprintf("%s the model is behind: the queue grew %d status lines in a row (%s)",
			c.yellow.Sprint(c.marks.warn), len(queues), strings.Join(lengths, " → ")),
		"  "+c.dim.Sprint("use a faster model or more workers"))
}

// nonLocalLlm warns that the LLM address is not on this machine.
func (c *console) nonLocalLlm(address string) {
	_ = c.block(c.errOut, // a lost line cannot be reported
		fmt.Sprintf("%s lm_studio.url is not on this machine: %s", c.yellow.Sprint(c.marks.warn), address),
		"  "+c.dim.Sprint("chat texts will leave this machine; use localhost to keep them here"))
}

// runStatus prints a status line at each tick until ctx ends, and the
// warning when the queue grew in behindLines lines in a row.
func (c *console) runStatus(ctx context.Context, ticks <-chan time.Time, cut func(time.Time) domain.Status) {
	var trend queueTrend
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticks:
			s := cut(now)
			c.status(s)
			if grown, ok := trend.add(s.Queue); ok {
				c.behind(grown)
			}
		}
	}
}

// queueTrend keeps the queue lengths of the last status lines while each is
// longer than the one before it.
type queueTrend struct{ lengths []int }

// add takes the next length; it gives the lengths and true once they grew
// behindLines times in a row, and then starts over.
func (t *queueTrend) add(length int) ([]int, bool) {
	if n := len(t.lengths); n > 0 && length <= t.lengths[n-1] {
		t.lengths = nil
	}
	t.lengths = append(t.lengths, length)
	if len(t.lengths) < behindLines {
		return nil, false
	}
	grown := t.lengths
	t.lengths = nil
	return grown, true
}

// tenths writes a duration in seconds with one decimal: 3.4s.
func tenths(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 1, 64) + "s"
}

// short writes a duration in whole seconds without zero tails: 5m, 4m35s, 8h33m.
func short(d time.Duration) string {
	text := d.Round(time.Second).String()
	if strings.HasSuffix(text, "m0s") {
		text = strings.TrimSuffix(text, "0s")
	}
	if strings.HasSuffix(text, "h0m") {
		text = strings.TrimSuffix(text, "0m")
	}
	return text
}
