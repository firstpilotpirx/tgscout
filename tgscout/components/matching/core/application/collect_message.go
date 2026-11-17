package application

import (
	"context"
	"slices"
	"strings"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// CollectMessage is the use case that drops what needs no judgment and puts
// the rest into the queue with the requests to judge it against.
type CollectMessage struct {
	settings SettingsRepository
	messages MessageRepository
	queue    MessageQueue
	tracker  *domain.MessageTracker
}

// NewCollectMessage builds the use case from its ports and the shared tracker.
func NewCollectMessage(settings SettingsRepository, messages MessageRepository, queue MessageQueue,
	tracker *domain.MessageTracker) *CollectMessage {
	return &CollectMessage{settings: settings, messages: messages, queue: queue, tracker: tracker}
}

// Collect drops a message with no text, of an unwatched chat, or seen
// already, and queues the rest. It blocks while the queue is full.
func (c *CollectMessage) Collect(ctx context.Context, message domain.Message) error {
	requests := c.settings.Settings().RequestsOf(message.Chat)
	if len(requests) == 0 || strings.TrimSpace(message.Text) == "" {
		return nil
	}
	if message.Edited {
		return c.collectEdit(ctx, message, requests)
	}
	if !c.tracker.Begin(message.Key) {
		return nil
	}
	processed, err := c.messages.IsProcessed(ctx, message.Key)
	if err != nil {
		c.tracker.Fail(message.Key)
		return err
	}
	if processed {
		c.tracker.Finish(message.Key)
		return nil
	}
	return c.queue.Put(ctx, domain.QueueItem{Message: message, Requests: requests})
}

func (c *CollectMessage) collectEdit(ctx context.Context, message domain.Message, requests []domain.Request) error {
	sent, err := c.messages.SentRequests(ctx, message.Key)
	if err != nil {
		return err
	}
	open := withoutSent(requests, sent)
	if len(open) == 0 {
		return nil
	}
	return c.queue.Put(ctx, domain.QueueItem{Message: message, Requests: open})
}

func withoutSent(requests []domain.Request, sent []string) []domain.Request {
	var open []domain.Request
	for _, request := range requests {
		if !slices.Contains(sent, request.ID) {
			open = append(open, request)
		}
	}
	return open
}
