package application_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

type matchFixture struct {
	repository *fakeMessageRepository
	llm        *fakeLlm
	notifier   *fakeNotifier
	tracker    *domain.MessageTracker
	useCase    *application.MatchMessage
}

func newMatch() matchFixture {
	f := matchFixture{
		repository: newRepository(),
		llm: &fakeLlm{answers: map[string]domain.Judgment{
			"a kids bike": {Matched: true, Raw: "yes"},
			"a desk lamp": {Raw: "no"},
		}},
		notifier: &fakeNotifier{},
		tracker:  domain.NewMessageTracker(),
	}
	f.useCase = application.NewMatchMessage(f.repository, f.llm, f.notifier, f.tracker)
	return f
}

func item(t *testing.T, id int) domain.QueueItem {
	return domain.QueueItem{Message: message(t, id, "Selling a bike"), Requests: twoRequests(t).Requests}
}

func (f matchFixture) begin(t *testing.T, item domain.QueueItem) domain.QueueItem {
	require.True(t, f.tracker.Begin(item.Message.Key))
	return item
}

func TestMatchJudgesAndSavesOncePerRequest(t *testing.T) {
	f := newMatch()

	_, err := f.useCase.Match(context.Background(), f.begin(t, item(t, 1)))

	require.NoError(t, err)
	assert.Equal(t, []string{"a kids bike", "a desk lamp"}, f.llm.calls)
	require.Len(t, f.repository.decisions, 2)
	assert.True(t, f.repository.decisions[0].Matched)
	assert.False(t, f.repository.decisions[1].Matched)
}

func TestMatchNotifiesAYesThenMarksItSent(t *testing.T) {
	f := newMatch()
	queued := f.begin(t, item(t, 1))

	_, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	require.Len(t, f.notifier.matches, 1)
	assert.Equal(t, "bike", f.notifier.matches[0].Request.ID)
	assert.Equal(t, []string{"bike"}, f.repository.sent[queued.Message.Key])
}

func TestMatchSendsAgainAYesSavedButNeverSent(t *testing.T) {
	f := newMatch()
	queued := f.begin(t, item(t, 1))
	f.repository.decisions = []domain.Decision{{Key: queued.Message.Key, RequestID: "bike", Matched: true}}

	_, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	assert.Contains(t, f.llm.calls, "a kids bike")
	require.Len(t, f.notifier.matches, 1)
}

func TestMatchProcessesAndMovesTheCursorAfterTheLastRequest(t *testing.T) {
	f := newMatch()
	queued := f.begin(t, item(t, 4))

	_, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	assert.True(t, f.repository.processed[queued.Message.Key])
	assert.Equal(t, 4, f.repository.cursor[7])
}

func TestMatchLeavesTheMessageUnprocessedWhenTheLlmFails(t *testing.T) {
	f := newMatch()
	f.llm.err = errBoom
	queued := f.begin(t, item(t, 4))

	_, err := f.useCase.Match(context.Background(), queued)

	var failure *application.MatchError
	require.ErrorAs(t, err, &failure)
	assert.Equal(t, application.StageLlm, failure.Stage)
	assert.Equal(t, queued.Message.Key, failure.Key)
	assert.ErrorIs(t, err, errBoom)
	assert.False(t, f.repository.processed[queued.Message.Key])
	assert.Zero(t, f.repository.cursor[7])
	assert.False(t, f.tracker.Begin(queued.Message.Key), "a failed key stays in flight")
}

func TestMatchLeavesTheMessageUnprocessedAndUnsentWhenNotifyFails(t *testing.T) {
	f := newMatch()
	f.notifier.err = errBoom
	queued := f.begin(t, item(t, 4))

	_, err := f.useCase.Match(context.Background(), queued)

	var failure *application.MatchError
	require.ErrorAs(t, err, &failure)
	assert.Equal(t, application.StageNotify, failure.Stage)
	assert.Empty(t, f.repository.sent[queued.Message.Key])
	assert.False(t, f.repository.processed[queued.Message.Key])
	assert.Zero(t, f.repository.cursor[7])
}

func TestMatchSavesAnUnclearJudgmentAsNoAndListsIt(t *testing.T) {
	f := newMatch()
	f.llm.answers["a kids bike"] = domain.Judgment{Unclear: true, Raw: "maybe"}
	queued := f.begin(t, item(t, 1))

	outcome, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	assert.False(t, f.repository.decisions[0].Matched)
	assert.Empty(t, f.notifier.matches)
	require.Len(t, outcome.Unclear, 1)
	assert.Equal(t, "bike", outcome.Unclear[0].Request.ID)
	assert.Equal(t, "maybe", outcome.Unclear[0].Raw)
}

func TestMatchOfAnEditSkipsASentYesAndNeverMovesTheCursor(t *testing.T) {
	f := newMatch()
	queued := item(t, 4)
	queued.Message.Edited = true
	f.repository.sent[queued.Message.Key] = []string{"bike"}

	_, err := f.useCase.Match(context.Background(), queued)

	require.NoError(t, err)
	assert.Equal(t, []string{"a desk lamp"}, f.llm.calls)
	assert.Empty(t, f.notifier.matches)
	assert.True(t, f.repository.processed[queued.Message.Key])
	assert.Zero(t, f.repository.cursor[7])
}

func TestMatchMovesTheCursorAsTheTrackerSaysOnOutOfOrderFinishes(t *testing.T) {
	f := newMatch()
	first, second, third := f.begin(t, item(t, 10)), f.begin(t, item(t, 11)), f.begin(t, item(t, 12))

	_, _ = f.useCase.Match(context.Background(), first)
	assert.Equal(t, 10, f.repository.cursor[7])
	_, _ = f.useCase.Match(context.Background(), third)
	assert.Equal(t, 10, f.repository.cursor[7])
	_, _ = f.useCase.Match(context.Background(), second)
	assert.Equal(t, 12, f.repository.cursor[7])
}

func TestMatchJudgesAnEditAndItsOriginalOneAtATime(t *testing.T) {
	f := newMatch()
	original := f.begin(t, item(t, 1))
	edit := item(t, 1)
	edit.Message.Edited = true

	var wait sync.WaitGroup
	for _, queued := range []domain.QueueItem{original, edit} {
		wait.Add(1)
		go func(queued domain.QueueItem) {
			defer wait.Done()
			_, _ = f.useCase.Match(context.Background(), queued)
		}(queued)
	}
	wait.Wait()

	assert.Len(t, f.notifier.matches, 1, "a sent yes is never sent twice")
}

// stoppedNotifier stands for a send Telegram never confirmed: it waits for the stop.
type stoppedNotifier struct{}

func (stoppedNotifier) Notify(ctx context.Context, _ domain.Match) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestMatchStoppedDuringTheSendLeavesTheMessageUnprocessed(t *testing.T) {
	f := newMatch()
	useCase := application.NewMatchMessage(f.repository, f.llm, stoppedNotifier{}, f.tracker)
	queued := f.begin(t, item(t, 5))
	ctx, stop := context.WithCancel(context.Background())
	stop()

	_, err := useCase.Match(ctx, queued)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, f.repository.sent[queued.Message.Key])
	assert.False(t, f.repository.processed[queued.Message.Key])
	assert.Zero(t, f.repository.cursor[7])
}
