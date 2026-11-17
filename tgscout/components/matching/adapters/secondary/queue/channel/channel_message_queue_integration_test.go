//go:build integration

package channel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func item(id int) domain.QueueItem {
	return domain.QueueItem{Message: domain.Message{Key: domain.MessageKey{ChatID: 7, MessageID: id}}}
}

func TestQueueGivesItemsInOrder(t *testing.T) {
	queue := channel.NewChannelMessageQueue(3)
	ctx := context.Background()
	for id := 1; id <= 3; id++ {
		require.NoError(t, queue.Put(ctx, item(id)))
	}

	for id := 1; id <= 3; id++ {
		got, err := queue.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, id, got.Message.Key.MessageID)
	}
}

func TestPutBlocksWhileFullAndGoesOnWhenAPlaceFreesUp(t *testing.T) {
	queue := channel.NewChannelMessageQueue(1)
	ctx := context.Background()
	require.NoError(t, queue.Put(ctx, item(1)))

	done := make(chan error, 1)
	go func() { done <- queue.Put(ctx, item(2)) }()
	select {
	case <-done:
		t.Fatal("Put went on while the queue was full")
	case <-time.After(50 * time.Millisecond):
	}

	_, err := queue.Receive(ctx)
	require.NoError(t, err)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Put stayed blocked after a place freed up")
	}
}

func TestPutAndReceiveEndWithTheContext(t *testing.T) {
	queue := channel.NewChannelMessageQueue(1)
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, queue.Put(ctx, item(1)))
	cancel()

	assert.ErrorIs(t, queue.Put(ctx, item(2)), context.Canceled)
	_, err := channel.NewChannelMessageQueue(1).Receive(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

// blockedPut tries a Put on the full queue until the timeout and says whether the full queue was reported.
func blockedPut(queue *channel.ChannelMessageQueue, full <-chan int, id int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_ = queue.Put(ctx, item(id))
	select {
	case <-full:
		return true
	default:
		return false
	}
}

func TestAFullQueueIsReportedOncePerEpisode(t *testing.T) {
	queue := channel.NewChannelMessageQueue(1)
	full := make(chan int, 10)
	queue.OnFull = func(size int) { full <- size }
	ctx := context.Background()
	require.NoError(t, queue.Put(ctx, item(1)))
	assert.Equal(t, 1, queue.Len())

	assert.True(t, blockedPut(queue, full, 2), "the first wait on the full queue is reported")
	assert.False(t, blockedPut(queue, full, 3), "no second report while the queue was never empty")

	_, err := queue.Receive(ctx)
	require.NoError(t, err)
	assert.Zero(t, queue.Len())
	require.NoError(t, queue.Put(ctx, item(4)))
	assert.True(t, blockedPut(queue, full, 5), "reported again after the queue was empty")
}

func TestTheFullReportNamesTheSize(t *testing.T) {
	queue := channel.NewChannelMessageQueue(2)
	full := make(chan int, 1)
	queue.OnFull = func(size int) { full <- size }
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, queue.Put(ctx, item(1)))
	require.NoError(t, queue.Put(ctx, item(2)))

	done := make(chan error, 1)
	go func() { done <- queue.Put(ctx, item(3)) }()

	select {
	case size := <-full:
		assert.Equal(t, 2, size)
	case <-time.After(time.Second):
		t.Fatal("the full queue was not reported")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}
