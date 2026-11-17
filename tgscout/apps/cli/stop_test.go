package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheFirstSignalStopsAndTheSecondExitsWith130(t *testing.T) {
	signals := make(chan os.Signal, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exited := make(chan int, 1)
	go stopOnSignals(signals, cancel, func(code int) { exited <- code })

	signals <- os.Interrupt
	<-ctx.Done()
	select {
	case code := <-exited:
		t.Fatalf("the first signal exited with %d", code)
	case <-time.After(20 * time.Millisecond):
	}
	signals <- os.Interrupt

	assert.Equal(t, 130, <-exited)
}

// runCtxClient records whether its run context was still alive when the work ended.
type runCtxClient struct {
	*fakeTelegram
	aliveAtEnd bool
}

func (c *runCtxClient) Run(ctx context.Context, fn func(context.Context) error) error {
	err := fn(ctx)
	c.aliveAtEnd = ctx.Err() == nil
	return err
}

func TestTheTelegramRunOutlivesTheStopUntilTheWorkEnds(t *testing.T) {
	client := &runCtxClient{fakeTelegram: &fakeTelegram{}}
	stop, cancel := context.WithCancel(context.Background())

	err := runDetached(stop, client, func(work context.Context) error {
		cancel()
		<-work.Done() // the work sees the stop
		return nil
	})

	require.NoError(t, err)
	assert.True(t, client.aliveAtEnd, "a send in work can still reach Telegram during the grace time")
}
