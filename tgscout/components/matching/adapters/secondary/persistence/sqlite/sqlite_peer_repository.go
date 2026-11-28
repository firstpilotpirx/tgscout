package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// SqlitePeerRepository keeps the found chats and the owner in known_peer.
type SqlitePeerRepository struct {
	db  *sql.DB
	now func() time.Time
}

// NewSqlitePeerRepository wraps an open, migrated database.
func NewSqlitePeerRepository(db *sql.DB) *SqlitePeerRepository {
	return &SqlitePeerRepository{db: db, now: time.Now}
}

// FindByRef reads the stored peer of the role and config text; false when none.
func (r *SqlitePeerRepository) FindByRef(ctx context.Context, role domain.PeerRole,
	ref string) (domain.KnownPeer, bool, error) {
	peer := domain.KnownPeer{Role: role, Ref: ref}
	var kind string
	err := r.db.QueryRowContext(ctx, `SELECT kind, peer_id, access_hash, username, title FROM known_peer
		WHERE role = ? AND ref = ?`, string(role), ref).
		Scan(&kind, &peer.PeerID, &peer.AccessHash, &peer.Username, &peer.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.KnownPeer{}, false, nil
	}
	if err != nil {
		return domain.KnownPeer{}, false, fmt.Errorf("read stored %s %s: %w", role, ref, err)
	}
	peer.Kind = domain.PeerKind(kind)
	return peer, true, nil
}

// Save inserts or replaces the peer of its role and config text.
func (r *SqlitePeerRepository) Save(ctx context.Context, peer domain.KnownPeer) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO known_peer
		(role, ref, kind, peer_id, access_hash, username, title, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (role, ref) DO UPDATE SET kind = excluded.kind, peer_id = excluded.peer_id,
			access_hash = excluded.access_hash, username = excluded.username, title = excluded.title,
			resolved_at = excluded.resolved_at`,
		string(peer.Role), peer.Ref, string(peer.Kind), peer.PeerID, peer.AccessHash, peer.Username, peer.Title,
		r.now().UTC())
	if err != nil {
		return fmt.Errorf("store %s %s: %w", peer.Role, peer.Ref, err)
	}
	return nil
}

// DeleteByRef drops the stored peer of the role and config text, if any.
func (r *SqlitePeerRepository) DeleteByRef(ctx context.Context, role domain.PeerRole, ref string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM known_peer WHERE role = ? AND ref = ?", string(role), ref); err != nil {
		return fmt.Errorf("drop stored %s %s: %w", role, ref, err)
	}
	return nil
}
