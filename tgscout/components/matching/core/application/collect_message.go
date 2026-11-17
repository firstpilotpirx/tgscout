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
	stats    *domain.RunStats
}

// NewCollectMessage builds the use case from its ports and the shared tracker.
func NewCollectMessage(settings SettingsRepository, messages MessageRepository, queue MessageQueue,
	tracker *domain.MessageTracker) *CollectMessage {
	return &CollectMessage{settings: settings, messages: messages, queue: queue, tracker: tracker}
}

// Counting makes the use case count each live message it queues in stats.
func (c *CollectMessage) Counting(stats *domain.RunStats) { c.stats = stats }

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
	message.Live = true
	_, err := c.admit(ctx, domain.QueueItem{Message: message, Requests: requests}, c.stats.In)
	return err
}

// CollectHistory takes a history message with the requests whose depth
// covers it through the same checks, and counts it in the catch-up tally.
// True means it went to the judge.
func (c *CollectMessage) CollectHistory(ctx context.Context, message domain.Message, requests []domain.Request) (bool, error) {
	tally := c.tracker.Tally()
	tally.Read(1)
	if len(requests) == 0 || strings.TrimSpace(message.Text) == "" {
		tally.Skip()
		return false, nil
	}
	queued, err := c.admit(ctx, domain.QueueItem{Message: message, Requests: requests},
		func() { tally.Expect(message.Key) })
	switch {
	case err != nil && queued:
		tally.Settle(message.Key, 0, true)
	case err != nil:
		tally.Fail()
	case !queued:
		tally.Skip()
	}
	return queued && err == nil, err
}

// admit begins a new message, drops one in flight or processed already, and
// queues the rest; expect runs before the put. True means it was handed on.
func (c *CollectMessage) admit(ctx context.Context, item domain.QueueItem, expect func()) (bool, error) {
	key := item.Message.Key
	if !c.tracker.Begin(key) {
		return false, nil
	}
	processed, err := c.messages.IsProcessed(ctx, key)
	if err != nil {
		c.tracker.Fail(key)
		return false, err
	}
	if processed {
		c.tracker.Finish(key)
		return false, nil
	}
	expect()
	return true, c.queue.Put(ctx, item)
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
