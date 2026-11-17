package application

import (
	"context"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Llm is the driven port that asks the LLM whether a message matches a request.
type Llm interface {
	// Judge asks once about one request (find) and one message text.
	Judge(ctx context.Context, find, text string) (domain.Judgment, error)
}
