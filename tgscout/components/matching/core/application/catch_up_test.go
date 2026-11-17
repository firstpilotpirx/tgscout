package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

var catchUpStart = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

type catchUpFixture struct {
	collectFixture
	catchUp *application.CatchUp
}

// depthRequests: "week" reads 100 messages and 7 days, "many" 2000 and 3.
func depthRequests(t *testing.T) domain.Settings {
	market := ref(t, "@made_up_market")
	return domain.Settings{Requests: []domain.Request{
		{ID: "week", Find: "a kids bike", Chats: []domain.RequestChat{{Ref: market, CatchUp: domain.CatchUp{Messages: 100, Days: 7}}}},
		{ID: "many", Find: "a desk lamp", Chats: []domain.RequestChat{{Ref: market, CatchUp: domain.CatchUp{Messages: 2000, Days: 3}}}},
	}}
}

func newCatchUp(t *testing.T) catchUpFixture {
	f := catchUpFixture{collectFixture: collectFixture{repository: newRepository(), queue: &fakeMessageQueue{},
		tracker: domain.NewMessageTracker()}}
	settings := fakeSettingsRepository{settings: depthRequests(t)}
	f.useCase = application.NewCollectMessage(settings, f.repository, f.queue, f.tracker)
	f.catchUp = application.NewCatchUp(settings, f.repository, f.useCase, f.tracker, func() time.Time { return catchUpStart })
	return f
}

func historyMessage(t *testing.T, id int, age time.Duration) domain.Message {
	each := message(t, id, "Selling a bike")
	each.SentAt = catchUpStart.Add(-age)
	return each
}

func TestOpenGivesTheWindowOfAFirstStart(t *testing.T) {
	f := newCatchUp(t)

	chat, err := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))

	require.NoError(t, err)
	assert.False(t, chat.HasCursor)
	assert.Equal(t, 0, chat.After)
	assert.Equal(t, 2000, chat.Depth.Messages)
	assert.Equal(t, 7, chat.Depth.Days)
	assert.Equal(t, catchUpStart.Add(-7*24*time.Hour), chat.Since())
}

func TestOpenStartsAfterTheStoredCursor(t *testing.T) {
	f := newCatchUp(t)
	at := catchUpStart.Add(-12 * 24 * time.Hour)
	f.repository.cursor[7] = 100
	f.repository.cursorAt[7] = at

	chat, err := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))

	require.NoError(t, err)
	assert.True(t, chat.HasCursor)
	assert.Equal(t, 100, chat.After)
	assert.Equal(t, at, chat.CursorAt)
}

func TestOpenReportsARepositoryError(t *testing.T) {
	f := newCatchUp(t)
	f.repository.err = errBoom

	_, err := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))

	assert.ErrorIs(t, err, errBoom)
}

func TestCollectQueuesAHistoryMessageWithTheRequestsThatCoverIt(t *testing.T) {
	f := newCatchUp(t)
	chat, _ := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))

	require.NoError(t, f.catchUp.Collect(context.Background(), chat, historyMessage(t, 1, 5*24*time.Hour), 50))
	require.NoError(t, f.catchUp.Collect(context.Background(), chat, historyMessage(t, 2, time.Hour), 150))
	require.NoError(t, f.catchUp.Collect(context.Background(), chat, historyMessage(t, 3, 5*24*time.Hour), 150))

	require.Len(t, f.queue.items, 2)
	assert.Equal(t, []string{"week"}, requestIDs(f.queue.items[0]))
	assert.Equal(t, []string{"many"}, requestIDs(f.queue.items[1]))
}

func requestIDs(item domain.QueueItem) []string {
	var ids []string
	for _, request := range item.Requests {
		ids = append(ids, request.ID)
	}
	return ids
}

func TestCollectSkipsWhatNeedsNoJudgment(t *testing.T) {
	f := newCatchUp(t)
	chat, _ := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))
	f.repository.processed[domain.MessageKey{ChatID: 7, MessageID: 1}] = true
	f.tracker.Begin(domain.MessageKey{ChatID: 7, MessageID: 2}) // also arrived as a new message
	blank := historyMessage(t, 3, time.Hour)
	blank.Text = " "

	for _, each := range []domain.Message{historyMessage(t, 1, time.Hour), historyMessage(t, 2, time.Hour), blank} {
		require.NoError(t, f.catchUp.Collect(context.Background(), chat, each, 1))
	}

	assert.Empty(t, f.queue.items)
	counts, err := f.catchUp.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, domain.CatchUpCounts{Read: 3, Skipped: 3}, counts)
}

func TestWaitGivesTheCountsOnceTheJudgeSettledEveryMessage(t *testing.T) {
	f := newCatchUp(t)
	chat, _ := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))
	require.NoError(t, f.catchUp.Collect(context.Background(), chat, historyMessage(t, 1, time.Hour), 1))
	match := application.NewMatchMessage(f.repository, &fakeLlm{answers: map[string]domain.Judgment{
		"a kids bike": {Matched: true, Raw: "yes"}}}, &fakeNotifier{}, f.tracker, application.Delivery{})

	_, err := match.Match(context.Background(), f.queue.items[0])
	require.NoError(t, err)

	counts, err := f.catchUp.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, domain.CatchUpCounts{Read: 1, Judged: 1, Matches: 1}, counts)
}

func TestCollectCountsARepositoryErrorAsAnError(t *testing.T) {
	f := newCatchUp(t)
	chat, _ := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))
	f.repository.err = errBoom

	err := f.catchUp.Collect(context.Background(), chat, historyMessage(t, 1, time.Hour), 1)

	assert.ErrorIs(t, err, errBoom)
	counts, _ := f.catchUp.Wait(context.Background())
	assert.Equal(t, 1, counts.Errors)
}

func TestTheCursorWaitsUntilTheChatIsClosed(t *testing.T) {
	f := newCatchUp(t)
	chat, _ := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))
	match := application.NewMatchMessage(f.repository, &fakeLlm{}, &fakeNotifier{}, f.tracker, application.Delivery{})
	live := domain.QueueItem{Message: message(t, 50, "Selling a lamp"), Requests: depthRequests(t).Requests}
	require.True(t, f.tracker.Begin(live.Message.Key))

	_, err := match.Match(context.Background(), live)
	require.NoError(t, err)
	assert.Equal(t, 0, f.repository.cursor[7])

	require.NoError(t, f.catchUp.Close(context.Background(), chat))
	assert.Equal(t, 50, f.repository.cursor[7])
}

func TestMatchSettlesOnlyAnExpectedNewMessage(t *testing.T) {
	f := newCatchUp(t)
	chat, _ := f.catchUp.Open(context.Background(), 7, ref(t, "@made_up_market"))
	require.NoError(t, f.catchUp.Collect(context.Background(), chat, historyMessage(t, 1, time.Hour), 1))
	match := application.NewMatchMessage(f.repository, &fakeLlm{}, &fakeNotifier{}, f.tracker, application.Delivery{})
	f.repository.err = errBoom // a failed database read ends the judgment
	edit := f.queue.items[0]
	edit.Message.Edited = true

	_, err := match.Match(context.Background(), edit)
	require.Error(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = f.catchUp.Wait(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded, "an edit does not settle the history message")

	_, err = match.Match(context.Background(), f.queue.items[0])
	require.Error(t, err)
	counts, err := f.catchUp.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, counts.Errors)
}
