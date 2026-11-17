//go:build integration

package telegram_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// playingClient plays its updates through Listen, then waits for the stop.
type playingClient struct {
	telegram.Client
	updates []telegram.Update
}

func (c playingClient) Listen(ctx context.Context, f func(context.Context, telegram.Update)) error {
	for _, update := range c.updates {
		f(ctx, update)
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestCollectorListensUntilTheStop(t *testing.T) {
	useCase := &recordingCollector{}
	collector := collect.NewTelegramCollector(chats(t), useCase, func(domain.Message, error) {})
	client := playingClient{updates: []telegram.Update{
		{ChatID: 42, MessageID: 1, Text: "Selling a bike"},
		{ChatID: 42, MessageID: 2, Service: true},
		{ChatID: 1234567890, MessageID: 3, Text: "Desk lamp"},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := collector.Listen(ctx, client)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Len(t, useCase.messages, 2)
	assert.Equal(t, 3, useCase.messages[1].Key.MessageID)
}
