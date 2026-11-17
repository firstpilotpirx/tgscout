package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func depthChat(t *testing.T, text string, messages, days int) domain.RequestChat {
	t.Helper()
	each := chat(t, text)
	each.CatchUp = domain.CatchUp{Messages: messages, Days: days}
	return each
}

func twoDepths(t *testing.T) domain.Settings {
	t.Helper()
	return domain.Settings{Requests: []domain.Request{
		{ID: "week", Chats: []domain.RequestChat{depthChat(t, "@made_up_market", 100, 7)}},
		{ID: "many", Chats: []domain.RequestChat{depthChat(t, "@made_up_market", 2000, 3)}},
		{ID: "other", Chats: []domain.RequestChat{depthChat(t, "@made_up_street", 9000, 30)}},
	}}
}

func TestCatchUpWindowTakesTheLargerDepthOfEachField(t *testing.T) {
	ref, _ := domain.NewChatRef("@made_up_market")

	assert.Equal(t, domain.CatchUp{Messages: 2000, Days: 7}, twoDepths(t).CatchUpWindow(ref))
}

func TestCatchUpWindowOfAnUnwatchedChatIsZero(t *testing.T) {
	ref, _ := domain.NewChatRef("@made_up_nobody")

	assert.Equal(t, domain.CatchUp{}, twoDepths(t).CatchUpWindow(ref))
}

func ids(requests []domain.Request) []string {
	var out []string
	for _, request := range requests {
		out = append(out, request.ID)
	}
	return out
}

func TestCoveringRequestsChecksEachRequestsOwnDepth(t *testing.T) {
	settings := twoDepths(t)
	ref, _ := domain.NewChatRef("@made_up_market")
	start := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	assert.Equal(t, []string{"week", "many"}, ids(settings.CoveringRequests(ref, 50, start.Add(-day), start)))
	assert.Equal(t, []string{"week"}, ids(settings.CoveringRequests(ref, 50, start.Add(-5*day), start)))
	assert.Equal(t, []string{"many"}, ids(settings.CoveringRequests(ref, 150, start.Add(-day), start)))
	assert.Empty(t, settings.CoveringRequests(ref, 150, start.Add(-5*day), start))
	assert.Equal(t, []string{"week", "many"}, ids(settings.CoveringRequests(ref, 100, start.Add(-3*day), start)))
}

func TestTallyCountsSettledMessages(t *testing.T) {
	tally := domain.NewCatchUpTally()
	tally.Read(4)
	tally.Skip()
	tally.Expect(key(1, 10))
	tally.Expect(key(1, 11))
	tally.Expect(key(1, 12))

	tally.Settle(key(1, 10), 2, false)
	tally.Settle(key(1, 11), 0, true)
	tally.Settle(key(1, 12), 0, false)
	tally.Settle(key(1, 99), 1, false) // not expected: a live message

	counts, err := tally.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, domain.CatchUpCounts{Read: 4, Skipped: 1, Judged: 2, Matches: 2, Errors: 1}, counts)
}

func TestTallySettlesAMessageOnce(t *testing.T) {
	tally := domain.NewCatchUpTally()
	tally.Expect(key(1, 10))

	tally.Settle(key(1, 10), 1, false)
	tally.Settle(key(1, 10), 1, false)

	counts, _ := tally.Wait(context.Background())
	assert.Equal(t, 1, counts.Judged)
	assert.Equal(t, 1, counts.Matches)
}

func TestTallyWaitsForEveryExpectedMessage(t *testing.T) {
	tally := domain.NewCatchUpTally()
	tally.Expect(key(1, 10))
	done := make(chan domain.CatchUpCounts)
	go func() {
		counts, _ := tally.Wait(context.Background())
		done <- counts
	}()

	select {
	case <-done:
		t.Fatal("Wait returned before the message was settled")
	case <-time.After(50 * time.Millisecond):
	}
	tally.Settle(key(1, 10), 0, false)
	assert.Equal(t, 1, (<-done).Judged)
}

func TestTallyWaitEndsWithTheContext(t *testing.T) {
	tally := domain.NewCatchUpTally()
	tally.Expect(key(1, 10))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tally.Wait(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestHeldCursorDoesNotMoveUntilReleased(t *testing.T) {
	cursor := domain.NewCursor()
	cursor.Hold()
	cursor.Begin(200)

	_, moved := cursor.Finish(200)
	assert.False(t, moved)

	cursor.Begin(150)
	cursor.Begin(160)
	position, moved := cursor.Release()
	assert.False(t, moved)
	assert.Equal(t, 0, position)

	position, moved = cursor.Finish(150)
	assert.True(t, moved)
	assert.Equal(t, 150, position)
	position, moved = cursor.Finish(160)
	assert.True(t, moved)
	assert.Equal(t, 200, position)
}

func TestReleaseMovesPastWhatFinishedWhileHeld(t *testing.T) {
	cursor := domain.NewCursor()
	cursor.Hold()
	cursor.Begin(5)
	cursor.Finish(5)

	position, moved := cursor.Release()

	assert.True(t, moved)
	assert.Equal(t, 5, position)
}

func TestTrackerHoldsOneChatOnly(t *testing.T) {
	tracker := domain.NewMessageTracker()
	tracker.Hold(1)
	tracker.Begin(key(1, 10))
	tracker.Begin(key(2, 10))

	_, moved := tracker.Finish(key(1, 10))
	assert.False(t, moved)
	_, moved = tracker.Finish(key(2, 10))
	assert.True(t, moved)

	position, moved := tracker.Release(1)
	assert.True(t, moved)
	assert.Equal(t, 10, position)
}
