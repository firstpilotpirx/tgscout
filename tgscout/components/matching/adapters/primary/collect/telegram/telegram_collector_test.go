package telegram_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

type recordingCollector struct {
	messages []domain.Message
	err      error
}

func (r *recordingCollector) Collect(_ context.Context, message domain.Message) error {
	r.messages = append(r.messages, message)
	return r.err
}

func chats(t *testing.T) map[int64]collect.Chat {
	market, err := domain.NewChatRef("@made_up_market")
	require.NoError(t, err)
	private, err := domain.NewChatRef("-1001234567890")
	require.NoError(t, err)
	return map[int64]collect.Chat{
		42:         {Ref: market, Username: "made_up_market"},
		1234567890: {Ref: private},
	}
}

func handle(t *testing.T, updates ...telegram.Update) (*recordingCollector, []error) {
	useCase := &recordingCollector{}
	var reported []error
	collector := collect.NewTelegramCollector(chats(t), useCase, func(_ domain.Message, err error) { reported = append(reported, err) })
	for _, update := range updates {
		collector.Handle(context.Background(), update)
	}
	return useCase, reported
}

func TestLinkOfAPublicChatUsesItsUsername(t *testing.T) {
	useCase, _ := handle(t, telegram.Update{ChatID: 42, MessageID: 4211, Text: "Selling a bike"})

	require.Len(t, useCase.messages, 1)
	assert.Equal(t, "https://t.me/made_up_market/4211", useCase.messages[0].Link)
	assert.Equal(t, "@made_up_market", useCase.messages[0].Chat.String())
	assert.Equal(t, domain.MessageKey{ChatID: 42, MessageID: 4211}, useCase.messages[0].Key)
}

func TestLinkOfAPrivateChatUsesItsID(t *testing.T) {
	useCase, _ := handle(t, telegram.Update{ChatID: 1234567890, MessageID: 98, Text: "Desk lamp"})

	assert.Equal(t, "https://t.me/c/1234567890/98", useCase.messages[0].Link)
}

func TestAnUpdateOfAnUnwatchedChatIsDropped(t *testing.T) {
	useCase, _ := handle(t, telegram.Update{ChatID: 7, MessageID: 1, Text: "Selling a bike"})

	assert.Empty(t, useCase.messages)
}

func TestAServiceUpdateIsDropped(t *testing.T) {
	useCase, _ := handle(t, telegram.Update{ChatID: 42, MessageID: 1, Service: true})

	assert.Empty(t, useCase.messages)
}

func TestAnEditIsPassedAsEdited(t *testing.T) {
	useCase, _ := handle(t, telegram.Update{ChatID: 42, MessageID: 1, Text: "Selling a red bike", Edited: true})

	assert.True(t, useCase.messages[0].Edited)
}

func TestACollectErrorIsReported(t *testing.T) {
	useCase := &recordingCollector{err: assert.AnError}
	var reported []error
	collector := collect.NewTelegramCollector(chats(t), useCase, func(_ domain.Message, err error) { reported = append(reported, err) })

	collector.Handle(context.Background(), telegram.Update{ChatID: 42, MessageID: 1, Text: "Selling a bike"})

	assert.Equal(t, []error{assert.AnError}, reported)
}
