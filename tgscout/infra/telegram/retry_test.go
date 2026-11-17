package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryWaitsEachFloodWaitAndCallsAgain(t *testing.T) {
	var reported, slept []time.Duration
	answers := []error{&FloodWait{Wait: 5 * time.Second}, &FloodWait{Wait: time.Hour}, nil}
	calls := 0

	value, err := Retry(context.Background(), func(d time.Duration) { reported = append(reported, d) },
		func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil },
		func() (int, error) { calls++; return calls, answers[calls-1] })

	require.NoError(t, err)
	assert.Equal(t, 3, value)
	assert.Equal(t, []time.Duration{5 * time.Second, time.Hour}, reported)
	assert.Equal(t, []time.Duration{5 * time.Second, time.Hour}, slept)
}

func TestRetryReturnsAnyOtherError(t *testing.T) {
	refused := errors.New("refused")
	calls := 0

	_, err := Retry(context.Background(), func(time.Duration) {},
		func(context.Context, time.Duration) error { return nil },
		func() (struct{}, error) { calls++; return struct{}{}, refused })

	require.ErrorIs(t, err, refused)
	assert.Equal(t, 1, calls)
}

func TestRetryStopsWhenTheWaitIsStopped(t *testing.T) {
	calls := 0

	_, err := Retry(context.Background(), func(time.Duration) {},
		func(context.Context, time.Duration) error { return context.Canceled },
		func() (struct{}, error) { calls++; return struct{}{}, &FloodWait{Wait: time.Second} })

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}
