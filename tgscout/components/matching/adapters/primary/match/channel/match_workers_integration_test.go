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

type scriptedMatcher struct{}

func (scriptedMatcher) Match(_ context.Context, item domain.QueueItem) (application.Outcome, error) {
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
	done := make(chan struct{})

	go func() {
		match.RunWorkers(ctx, 2, items, scriptedMatcher{}, reporter)
		close(done)
	}()
	assert.Eventually(t, func() bool { u, f := reporter.counts(); return u == 1 && f == 1 }, time.Second, 10*time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers did not stop with the context")
	}
	assert.Equal(t, []string{"maybe"}, reporter.unclear)
	assert.Equal(t, []int{2}, reporter.failed)
}
