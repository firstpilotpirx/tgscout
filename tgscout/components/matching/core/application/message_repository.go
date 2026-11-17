package application

import (
	"context"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// MessageRepository is the driven port that keeps processed messages,
// decisions and the cursor of each chat.
type MessageRepository interface {
	IsProcessed(ctx context.Context, key domain.MessageKey) (bool, error)
	// SentRequests lists the request IDs whose «yes» was sent for the message.
	SentRequests(ctx context.Context, key domain.MessageKey) ([]string, error)
	// SaveDecision inserts or replaces the decision of one message and request.
	SaveDecision(ctx context.Context, decision domain.Decision) error
	MarkSent(ctx context.Context, key domain.MessageKey, requestID string, at time.Time) error
	MarkProcessed(ctx context.Context, key domain.MessageKey, at time.Time) error
	// MoveCursor moves the chat's cursor forward; it never moves it back.
	MoveCursor(ctx context.Context, chatID int64, messageID int, at time.Time) error
}
