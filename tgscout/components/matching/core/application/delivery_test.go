package application_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// scriptedLlm fails the first calls with the given errors, then answers.
type scriptedLlm struct {
	mu     sync.Mutex
	fails  []error
	answer domain.Judgment
	calls  int
}

func (l *scriptedLlm) Judge(context.Context, string, string) (domain.Judgment, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if len(l.fails) > 0 {
		err := l.fails[0]
		l.fails = l.fails[1:]
		return domain.Judgment{}, err
	}
	return l.answer, nil
}

func failures(err error, n int) []error {
	errs := make([]error, n)
	for i := range errs {
		errs[i] = err
	}
	return errs
}

var errRejected = fmt.Errorf("answered 400 Bad Request: %w", application.ErrLlmRejected)

// recordingDelivery records the pauses and the reports instead of waiting.
type recordingDelivery struct {
	mu      sync.Mutex
	pauses  []time.Duration
	events  []string
	stopAt  int // the pause that ends with the stop; 0 never
	gaveUp  []string
	sendErr []error
}

func (d *recordingDelivery) delivery(maxPause time.Duration) application.Delivery {
	return application.Delivery{Backoff: domain.Backoff{Max: maxPause}, Sleep: d.sleep, Reporter: d}
}

func (d *recordingDelivery) sleep(_ context.Context, pause time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pauses = append(d.pauses, pause)
	if d.stopAt > 0 && len(d.pauses) == d.stopAt {
		return context.Canceled
	}
	return nil
}

func (d *recordingDelivery) LlmDown(time.Duration) { d.event("down") }
func (d *recordingDelivery) LlmBack()              { d.event("back") }
func (d *recordingDelivery) GaveUp(_ domain.QueueItem, request domain.Request, tries int, _ error) {
	d.event(fmt.Sprintf("gave up %s after %d", request.ID, tries))
}
func (d *recordingDelivery) SendFailed(_ domain.Match, err error, pause time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sendErr = append(d.sendErr, err)
	d.events = append(d.events, fmt.Sprintf("send failed, retry in %s", pause))
}

func (d *recordingDelivery) event(text string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, text)
}

// oneRequest is a queued message of the bike request only.
func oneRequestItem(t *testing.T, id int) domain.QueueItem {
	queued := item(t, id)
	queued.Requests = queued.Requests[:1]
	return queued
}

func newRetryMatch(llm application.Llm, notifier application.Notifier, delivery application.Delivery) matchFixture {
	f := newMatch()
	f.useCase = application.NewMatchMessage(f.repository, llm, notifier, f.tracker, delivery)
	return f
}

func TestTheLlmPausesGrowUpToTheCap(t *testing.T) {
	d := &recordingDelivery{}
	llm := &scriptedLlm{fails: failures(errBoom, 5), answer: domain.Judgment{Raw: "no"}}
	f := newRetryMatch(llm, f0(), d.delivery(4*time.Second))

	_, err := f.useCase.Match(context.Background(), f.begin(t, oneRequestItem(t, 1)))

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second, 4 * time.Second}, d.pauses)
	assert.Equal(t, 6, llm.calls)
}

func f0() *fakeNotifier { return &fakeNotifier{} }

func TestAMessageTheModelNeverManagesIsSavedAsNoAfterTenTries(t *testing.T) {
	d := &recordingDelivery{}
	llm := &scriptedLlm{fails: failures(errRejected, 20)}
	notifier := f0()
	f := newRetryMatch(llm, notifier, d.delivery(time.Minute))
	queued := f.begin(t, oneRequestItem(t, 3))

	_, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	assert.Equal(t, 10, llm.calls)
	assert.Len(t, d.pauses, 9)
	require.Len(t, f.repository.decisions, 1)
	assert.False(t, f.repository.decisions[0].Matched)
	assert.True(t, f.repository.processed[queued.Message.Key])
	assert.Equal(t, 3, f.repository.cursor[7])
	assert.Contains(t, d.events, "gave up bike after 10")
	assert.Empty(t, notifier.matches)
}

func TestNoAnswerNeverGivesUp(t *testing.T) {
	d := &recordingDelivery{}
	llm := &scriptedLlm{fails: failures(errBoom, 12), answer: domain.Judgment{Matched: true, Raw: "yes"}}
	notifier := f0()
	f := newRetryMatch(llm, notifier, d.delivery(time.Minute))

	_, err := f.useCase.Match(context.Background(), f.begin(t, oneRequestItem(t, 1)))

	require.NoError(t, err)
	require.Len(t, notifier.matches, 1)
	require.Len(t, f.repository.decisions, 1)
	assert.True(t, f.repository.decisions[0].Matched)
	assert.NotContains(t, d.events, "gave up bike after 10")
}

func TestOneLineWhenTheLlmGoesAndOneWhenItIsBack(t *testing.T) {
	d := &recordingDelivery{}
	llm := &scriptedLlm{fails: failures(errBoom, 3), answer: domain.Judgment{Raw: "no"}}
	f := newRetryMatch(llm, f0(), d.delivery(time.Minute))

	_, err := f.useCase.Match(context.Background(), f.begin(t, oneRequestItem(t, 1)))
	require.NoError(t, err)
	_, err = f.useCase.Match(context.Background(), f.begin(t, oneRequestItem(t, 2)))
	require.NoError(t, err)

	assert.Equal(t, []string{"down", "back"}, d.events)
}

func TestARejectedAnswerDoesNotSayTheLlmIsDown(t *testing.T) {
	d := &recordingDelivery{}
	llm := &scriptedLlm{fails: failures(errRejected, 2), answer: domain.Judgment{Raw: "no"}}
	f := newRetryMatch(llm, f0(), d.delivery(time.Minute))

	_, err := f.useCase.Match(context.Background(), f.begin(t, oneRequestItem(t, 1)))

	require.NoError(t, err)
	assert.Empty(t, d.events)
}

// flakyNotifier fails the first sends, then records the match.
type flakyNotifier struct {
	fails   int
	tries   int
	matches []domain.Match
}

func (n *flakyNotifier) Notify(_ context.Context, match domain.Match) error {
	n.tries++
	if n.tries <= n.fails {
		return errBoom
	}
	n.matches = append(n.matches, match)
	return nil
}

func TestASendThatFailsTwiceIsTriedAgainUntilItIsSent(t *testing.T) {
	d := &recordingDelivery{}
	notifier := &flakyNotifier{fails: 2}
	f := newRetryMatch(&scriptedLlm{answer: domain.Judgment{Matched: true, Raw: "yes"}}, notifier, d.delivery(time.Minute))
	queued := f.begin(t, oneRequestItem(t, 1))

	_, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	assert.Equal(t, []string{"send failed, retry in 1s", "send failed, retry in 2s"}, d.events)
	assert.Len(t, notifier.matches, 1)
	assert.Equal(t, []string{"bike"}, f.repository.sent[queued.Message.Key])
	assert.True(t, f.repository.processed[queued.Message.Key])
}

func TestAPauseThatEndsAtTheStopLeavesTheMessageUnprocessed(t *testing.T) {
	d := &recordingDelivery{stopAt: 2}
	f := newRetryMatch(&scriptedLlm{fails: failures(errBoom, 5)}, f0(), d.delivery(time.Minute))
	queued := f.begin(t, oneRequestItem(t, 4))

	_, err := f.useCase.Match(context.Background(), queued)

	require.ErrorIs(t, err, context.Canceled)
	var failure *application.MatchError
	require.ErrorAs(t, err, &failure)
	assert.Equal(t, application.StageLlm, failure.Stage)
	assert.False(t, f.repository.processed[queued.Message.Key])
	assert.Zero(t, f.repository.cursor[7])
	assert.Empty(t, f.repository.decisions)
}

func TestASendPauseThatEndsAtTheStopLeavesTheMessageUnsent(t *testing.T) {
	d := &recordingDelivery{stopAt: 1}
	f := newRetryMatch(&scriptedLlm{answer: domain.Judgment{Matched: true, Raw: "yes"}},
		&flakyNotifier{fails: 5}, d.delivery(time.Minute))
	queued := f.begin(t, oneRequestItem(t, 4))

	_, err := f.useCase.Match(context.Background(), queued)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, f.repository.sent[queued.Message.Key])
	assert.False(t, f.repository.processed[queued.Message.Key])
}

func TestAStopDuringTheLlmCallIsNotRetried(t *testing.T) {
	d := &recordingDelivery{}
	f := newRetryMatch(&scriptedLlm{fails: []error{context.Canceled}}, f0(), d.delivery(time.Minute))
	ctx, stop := context.WithCancel(context.Background())
	stop()

	_, err := f.useCase.Match(ctx, f.begin(t, oneRequestItem(t, 4)))

	require.True(t, errors.Is(err, context.Canceled))
	assert.Empty(t, d.pauses)
}
