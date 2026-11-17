package domain

import (
	"math"
	"slices"
	"sync"
	"time"
)

// P95 is the nearest-rank 95th percentile of the samples: the value at rank
// ⌈0.95·n⌉ of the sorted samples. False means there was no sample.
func P95(samples []time.Duration) (time.Duration, bool) {
	if len(samples) == 0 {
		return 0, false
	}
	sorted := slices.Clone(samples)
	slices.Sort(sorted)
	rank := int(math.Ceil(0.95 * float64(len(sorted))))
	return sorted[rank-1], true
}

// Status is what tgscout did over a span of the run: the live messages that
// came in, the messages judged, the matches sent, the queue length at its
// end and the p95 of the latency and of the LLM call time.
type Status struct {
	In, Judged, Matches, Queue int
	Took                       time.Duration
	Latency                    time.Duration
	HasLatency                 bool
	LlmCall                    time.Duration
	HasLlmCall                 bool
}

// tally is the counts and samples of one span.
type tally struct {
	in, judged, matches int
	latencies, calls    []time.Duration
}

func (t tally) status(queue int, took time.Duration) Status {
	status := Status{In: t.in, Judged: t.judged, Matches: t.matches, Queue: queue, Took: took}
	status.Latency, status.HasLatency = P95(t.latencies)
	status.LlmCall, status.HasLlmCall = P95(t.calls)
	return status
}

// RunStats counts the run, per interval and since start. The workers and the
// collector record at once, so a mutex guards it. Every method of a nil
// *RunStats does nothing.
type RunStats struct {
	mu            sync.Mutex
	start, cut    time.Time
	interval, run tally
}

// NewRunStats starts counting at start.
func NewRunStats(start time.Time) *RunStats {
	return &RunStats{start: start, cut: start}
}

// In counts a live message handed to the judge.
func (s *RunStats) In() {
	s.record(func(t *tally) { t.in++ })
}

// Judged counts a message whose judgment ended; latency is its time from the
// chat to the end of the judgment, nil when it does not count (not live).
func (s *RunStats) Judged(latency *time.Duration) {
	s.record(func(t *tally) {
		t.judged++
		if latency != nil {
			t.latencies = append(t.latencies, *latency)
		}
	})
}

// Matched counts a match whose send Telegram confirmed.
func (s *RunStats) Matched() {
	s.record(func(t *tally) { t.matches++ })
}

// LlmCall records how long one LLM call took.
func (s *RunStats) LlmCall(took time.Duration) {
	s.record(func(t *tally) { t.calls = append(t.calls, took) })
}

func (s *RunStats) record(f func(*tally)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f(&s.interval)
	f(&s.run)
}

// Cut gives the status of the interval since the last cut and starts a new one.
func (s *RunStats) Cut(queue int, now time.Time) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.interval.status(queue, now.Sub(s.cut))
	s.interval, s.cut = tally{}, now
	return status
}

// Total gives the status since start.
func (s *RunStats) Total(queue int, now time.Time) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.run.status(queue, now.Sub(s.start))
}
