package telegram

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pacerStart = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)

// madeUpClock is a clock that moves only when the pacer sleeps or a test
// says a call took some time.
type madeUpClock struct {
	mu     sync.Mutex
	at     time.Time
	sleeps []time.Duration
}

func (c *madeUpClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *madeUpClock) sleep(_ context.Context, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sleeps = append(c.sleeps, d)
	c.at = c.at.Add(d)
	return nil
}

func (c *madeUpClock) pass(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

func newMadeUpPacer(pause time.Duration) (*Pacer, *madeUpClock) {
	clock := &madeUpClock{at: pacerStart}
	return &Pacer{Pause: pause, now: clock.now, sleep: clock.sleep}, clock
}

// TestPacerKeepsThePauseBetweenCallStarts: history calls keep the pause.
func TestPacerKeepsThePauseBetweenCallStarts(t *testing.T) {
	pacer, clock := newMadeUpPacer(time.Second)
	ctx := context.Background()

	require.NoError(t, pacer.Wait(ctx))
	first := clock.now()
	require.NoError(t, pacer.Wait(ctx))
	second := clock.now()

	assert.Equal(t, first, pacerStart, "the first call starts at once")
	assert.GreaterOrEqual(t, second.Sub(first), time.Second)
}

func TestPacerCountsASlowCallIntoThePause(t *testing.T) {
	pacer, clock := newMadeUpPacer(time.Second)
	ctx := context.Background()

	require.NoError(t, pacer.Wait(ctx))
	clock.pass(400 * time.Millisecond) // the call took 0.4 s
	require.NoError(t, pacer.Wait(ctx))

	assert.Equal(t, []time.Duration{600 * time.Millisecond}, clock.sleeps)
	assert.Equal(t, pacerStart.Add(time.Second), clock.now())
}

// TestPacerSharedByTwoReaders: the gap read keeps the pause while catch-up reads.
func TestPacerSharedByTwoReaders(t *testing.T) {
	pacer, clock := newMadeUpPacer(time.Second)
	ctx := context.Background()
	var mu sync.Mutex
	var starts []time.Time
	var readers sync.WaitGroup
	for range 2 {
		readers.Go(func() {
			for range 3 {
				if pacer.Wait(ctx) != nil {
					return
				}
				mu.Lock()
				starts = append(starts, clock.now())
				mu.Unlock()
			}
		})
	}
	readers.Wait()

	require.Len(t, starts, 6)
	seen := map[time.Time]bool{}
	for _, start := range starts {
		assert.False(t, seen[start], "two calls started at %s", start)
		seen[start] = true
		offset := start.Sub(pacerStart)
		assert.Zero(t, offset%time.Second, "a call started %s after the first", offset)
	}
}

// TestPacerOffReturnsAtOnce: no pause when it is off.
func TestPacerOffReturnsAtOnce(t *testing.T) {
	pacer, clock := newMadeUpPacer(0)
	ctx := context.Background()

	for range 3 {
		require.NoError(t, pacer.Wait(ctx))
	}

	assert.Empty(t, clock.sleeps)
	assert.Equal(t, pacerStart, clock.now())
}

func TestPacerStopsWithTheContext(t *testing.T) {
	pacer := NewPacer(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, pacer.Wait(ctx))
	cancel()

	assert.ErrorIs(t, pacer.Wait(ctx), context.Canceled)
}
