package application

import (
	"context"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Notifier is the driven port that tells the owner about a match.
type Notifier interface {
	Notify(ctx context.Context, match domain.Match) error
}
