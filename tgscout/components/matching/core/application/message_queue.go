package application

import (
	"context"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// MessageQueue is the driven port that carries collected messages to the matcher workers.
type MessageQueue interface {
	// Put adds an item; it blocks while the queue is full, until ctx ends.
	Put(ctx context.Context, item domain.QueueItem) error
}
