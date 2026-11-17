// Package channel carries queued messages to the matcher workers in a buffered channel.
package channel

import (
	"context"
	"sync/atomic"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// ChannelMessageQueue holds at most its size of items waiting for the LLM.
type ChannelMessageQueue struct {
	// OnFull, when set, is called with the size the first time a Put waits on
	// the full queue, and again only after the queue was empty in between.
	OnFull func(size int)
	items  chan domain.QueueItem
	full   atomic.Bool
}

// NewChannelMessageQueue makes a queue of the given size (at least 1).
func NewChannelMessageQueue(size int) *ChannelMessageQueue {
	return &ChannelMessageQueue{items: make(chan domain.QueueItem, max(size, 1))}
}

// Len is the number of items waiting.
func (q *ChannelMessageQueue) Len() int { return len(q.items) }

// Put adds an item; it blocks while the queue is full, until ctx ends.
func (q *ChannelMessageQueue) Put(ctx context.Context, item domain.QueueItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case q.items <- item:
		return nil
	default:
	}
	if !q.full.Swap(true) && q.OnFull != nil {
		q.OnFull(cap(q.items))
	}
	select {
	case q.items <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Receive gives the next item, waiting until there is one or ctx ends.
func (q *ChannelMessageQueue) Receive(ctx context.Context) (domain.QueueItem, error) {
	if err := ctx.Err(); err != nil {
		return domain.QueueItem{}, err
	}
	select {
	case item := <-q.items:
		if len(q.items) == 0 {
			q.full.Store(false)
		}
		return item, nil
	case <-ctx.Done():
		return domain.QueueItem{}, ctx.Err()
	}
}
