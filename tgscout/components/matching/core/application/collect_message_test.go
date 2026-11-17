package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

type collectFixture struct {
	repository *fakeMessageRepository
	queue      *fakeMessageQueue
	tracker    *domain.MessageTracker
	useCase    *application.CollectMessage
}

func ref(t *testing.T, text string) domain.ChatRef {
	t.Helper()
	chat, err := domain.NewChatRef(text)
	require.NoError(t, err)
	return chat
}

func twoRequests(t *testing.T) domain.Settings {
	market := domain.RequestChat{Ref: ref(t, "@made_up_market")}
	return domain.Settings{Requests: []domain.Request{
		{ID: "bike", Find: "a kids bike", Chats: []domain.RequestChat{market}},
		{ID: "lamp", Find: "a desk lamp", Chats: []domain.RequestChat{market}},
	}}
}

func newCollect(t *testing.T) collectFixture {
	f := collectFixture{repository: newRepository(), queue: &fakeMessageQueue{}, tracker: domain.NewMessageTracker()}
	f.useCase = application.NewCollectMessage(fakeSettingsRepository{settings: twoRequests(t)}, f.repository, f.queue, f.tracker)
	return f
}

func message(t *testing.T, id int, text string) domain.Message {
	return domain.Message{Key: domain.MessageKey{ChatID: 7, MessageID: id}, Chat: ref(t, "@made_up_market"), Text: text}
}

func TestCollectQueuesANewMessageWithEveryRequestOfItsChat(t *testing.T) {
	f := newCollect(t)

	require.NoError(t, f.useCase.Collect(context.Background(), message(t, 1, "Selling a bike")))

	require.Len(t, f.queue.items, 1)
	assert.Equal(t, "bike", f.queue.items[0].Requests[0].ID)
	assert.Equal(t, "lamp", f.queue.items[0].Requests[1].ID)
}

func TestCollectDropsAMessageWithNoText(t *testing.T) {
	f := newCollect(t)

	require.NoError(t, f.useCase.Collect(context.Background(), message(t, 1, "  \n ")))

	assert.Empty(t, f.queue.items)
}

func TestCollectDropsAChatNoRequestWatches(t *testing.T) {
	f := newCollect(t)
	other := message(t, 1, "Selling a bike")
	other.Chat = ref(t, "@made_up_other")

	require.NoError(t, f.useCase.Collect(context.Background(), other))

	assert.Empty(t, f.queue.items)
}

func TestCollectDropsADuplicateInFlight(t *testing.T) {
	f := newCollect(t)

	require.NoError(t, f.useCase.Collect(context.Background(), message(t, 1, "Selling a bike")))
	require.NoError(t, f.useCase.Collect(context.Background(), message(t, 1, "Selling a bike")))

	assert.Len(t, f.queue.items, 1)
}

func TestCollectDropsAProcessedMessage(t *testing.T) {
	f := newCollect(t)
	f.repository.processed[domain.MessageKey{ChatID: 7, MessageID: 1}] = true

	require.NoError(t, f.useCase.Collect(context.Background(), message(t, 1, "Selling a bike")))

	assert.Empty(t, f.queue.items)
	assert.True(t, f.tracker.Begin(domain.MessageKey{ChatID: 7, MessageID: 1}), "the processed key leaves the in-flight set")
}

func TestCollectQueuesAnEditWithOnlyTheRequestsWithoutASentYes(t *testing.T) {
	f := newCollect(t)
	f.repository.sent[domain.MessageKey{ChatID: 7, MessageID: 1}] = []string{"bike"}
	edit := message(t, 1, "Selling a bike and a lamp")
	edit.Edited = true

	require.NoError(t, f.useCase.Collect(context.Background(), edit))

	require.Len(t, f.queue.items, 1)
	require.Len(t, f.queue.items[0].Requests, 1)
	assert.Equal(t, "lamp", f.queue.items[0].Requests[0].ID)
	assert.True(t, f.queue.items[0].Message.Edited)
}

func TestCollectDropsAnEditWhoseRequestsAllHaveASentYes(t *testing.T) {
	f := newCollect(t)
	f.repository.sent[domain.MessageKey{ChatID: 7, MessageID: 1}] = []string{"bike", "lamp"}
	edit := message(t, 1, "Selling a bike and a lamp")
	edit.Edited = true

	require.NoError(t, f.useCase.Collect(context.Background(), edit))

	assert.Empty(t, f.queue.items)
}

func TestCollectReportsARepositoryError(t *testing.T) {
	f := newCollect(t)
	f.repository.err = errBoom

	assert.ErrorIs(t, f.useCase.Collect(context.Background(), message(t, 1, "Selling a bike")), errBoom)
}
