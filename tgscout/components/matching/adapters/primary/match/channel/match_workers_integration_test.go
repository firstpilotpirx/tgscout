//go:build integration

package channel_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	match "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/match/channel"
	queue "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// scriptedMatcher answers per message ID and records the IDs it was given.
type scriptedMatcher struct {
	mu   sync.Mutex
	seen []int
}

func (m *scriptedMatcher) Match(_ context.Context, item domain.QueueItem) (application.Outcome, error) {
	m.mu.Lock()
	m.seen = append(m.seen, item.Message.Key.MessageID)
	m.mu.Unlock()
	switch item.Message.Key.MessageID {
	case 1:
		return application.Outcome{Unclear: []application.Unclear{{Request: domain.Request{ID: "bike"}, Raw: "maybe"}}}, nil
	case 2:
		return application.Outcome{}, &application.MatchError{Key: item.Message.Key, Stage: application.StageLlm, Err: errors.New("down")}
	}
	return application.Outcome{}, nil
}

type recordingReporter struct {
	mu      sync.Mutex
	unclear []string
	failed  []int
}

func (r *recordingReporter) Unclear(_ domain.QueueItem, unclear application.Unclear) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unclear = append(r.unclear, unclear.Raw)
}

func (r *recordingReporter) Failed(item domain.QueueItem, _ error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failed = append(r.failed, item.Message.Key.MessageID)
}

func (r *recordingReporter) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.unclear), len(r.failed)
}

func TestWorkersReportWarningsAndErrorsAndStopWithTheContext(t *testing.T) {
	items := queue.NewChannelMessageQueue(10)
	ctx, cancel := context.WithCancel(context.Background())
	for id := 1; id <= 3; id++ {
		require.NoError(t, items.Put(ctx, domain.QueueItem{Message: domain.Message{Key: domain.MessageKey{ChatID: 7, MessageID: id}}}))
	}
	reporter := &recordingReporter{}
	matcher := &scriptedMatcher{}
	done := make(chan struct{})

	go func() {
		match.RunWorkers(ctx, ctx, match.Pool{Workers: 1, Queue: items, Matcher: matcher, Report: reporter})
		close(done)
	}()
	assert.Eventually(t, func() bool {
		matcher.mu.Lock()
		defer matcher.mu.Unlock()
		u, f := reporter.counts()
		return u == 1 && f == 1 && len(matcher.seen) == 3
	}, time.Second, 10*time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers did not stop with the context")
	}
	assert.Equal(t, []string{"maybe"}, reporter.unclear)
	assert.Equal(t, []int{2}, reporter.failed)
	assert.Equal(t, []int{1, 2, 3}, matcher.seen, "the worker goes on with the next message after a failure")
}

// heldMatcher holds each Match until it is released or its context ends.
type heldMatcher struct {
	started chan int
	release chan struct{}
	ended   chan error
}

func newHeldMatcher() *heldMatcher {
	return &heldMatcher{started: make(chan int, 10), release: make(chan struct{}), ended: make(chan error, 10)}
}

func (m *heldMatcher) Match(ctx context.Context, item domain.QueueItem) (application.Outcome, error) {
	m.started <- item.Message.Key.MessageID
	select {
	case <-m.release:
		m.ended <- nil
		return application.Outcome{}, nil
	case <-ctx.Done():
		m.ended <- ctx.Err()
		return application.Outcome{}, ctx.Err()
	}
}

func queueOf(t *testing.T, ids ...int) *queue.ChannelMessageQueue {
	items := queue.NewChannelMessageQueue(10)
	for _, id := range ids {
		require.NoError(t, items.Put(context.Background(), domain.QueueItem{Message: domain.Message{Key: domain.MessageKey{ChatID: 7, MessageID: id}}}))
	}
	return items
}

func TestAStopLetsTheMessageInWorkFinishAndTakesNoMore(t *testing.T) {
	items := queueOf(t, 1, 2)
	matcher := newHeldMatcher()
	busy := &match.Busy{}
	stop, stopNow := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		match.RunWorkers(stop, context.Background(), match.Pool{Workers: 1, Queue: items, Matcher: matcher,
			Report: &recordingReporter{}, Busy: busy})
		close(done)
	}()
	assert.Equal(t, 1, <-matcher.started)
	assert.Equal(t, 1, busy.Count())

	stopNow()
	close(matcher.release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the workers did not end after the stop")
	}
	require.NoError(t, <-matcher.ended, "the message in work finished on the work context")
	assert.Zero(t, busy.Count())
	assert.Equal(t, 1, items.Len(), "no message is taken after the stop")
}

func TestAMatchInWorkEndsWithTheWorkContext(t *testing.T) {
	items := queueOf(t, 1)
	matcher := newHeldMatcher()
	reporter := &recordingReporter{}
	stop, stopNow := context.WithCancel(context.Background())
	work, endWork := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		match.RunWorkers(stop, work, match.Pool{Workers: 2, Queue: items, Matcher: matcher, Report: reporter})
		close(done)
	}()
	<-matcher.started

	stopNow()
	endWork()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the workers did not end with the work context")
	}
	require.ErrorIs(t, <-matcher.ended, context.Canceled)
	_, failed := reporter.counts()
	assert.Zero(t, failed, "a judgment the stop ended is not reported as failed")
}
