package application

import (
	"context"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// PeerRepository is the driven port that keeps the found chats and the owner
// by their config text.
type PeerRepository interface {
	// FindByRef reads the stored peer of the role and config text; false when none.
	FindByRef(ctx context.Context, role domain.PeerRole, ref string) (domain.KnownPeer, bool, error)
	// Save inserts or replaces the peer of its role and config text.
	Save(ctx context.Context, peer domain.KnownPeer) error
	// DeleteByRef drops the stored peer of the role and config text, if any.
	DeleteByRef(ctx context.Context, role domain.PeerRole, ref string) error
}
