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
