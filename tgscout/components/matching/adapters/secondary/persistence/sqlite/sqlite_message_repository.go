package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// SqliteMessageRepository keeps processed messages, decisions and cursors in
// the tables of the first migration.
type SqliteMessageRepository struct{ db *sql.DB }

// NewSqliteMessageRepository wraps an open, migrated database.
func NewSqliteMessageRepository(db *sql.DB) *SqliteMessageRepository {
	return &SqliteMessageRepository{db: db}
}

// IsProcessed tells whether the message was marked processed.
func (r *SqliteMessageRepository) IsProcessed(ctx context.Context, key domain.MessageKey) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx,
		"SELECT 1 FROM processed_message WHERE chat_id = ? AND message_id = ?", key.ChatID, key.MessageID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read processed mark: %w", err)
	}
	return true, nil
}

// SentRequests lists the requests whose «yes» was sent for the message.
func (r *SqliteMessageRepository) SentRequests(ctx context.Context, key domain.MessageKey) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT request_id FROM decision
		WHERE chat_id = ? AND message_id = ? AND sent_at IS NOT NULL ORDER BY request_id`, key.ChatID, key.MessageID)
	if err != nil {
		return nil, fmt.Errorf("read sent requests: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var sent []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read sent requests: %w", err)
		}
		sent = append(sent, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read sent requests: %w", err)
	}
	return sent, nil
}

// SaveDecision inserts the decision or replaces the one of the same message
// and request; a sent mark stays.
func (r *SqliteMessageRepository) SaveDecision(ctx context.Context, decision domain.Decision) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO decision (chat_id, message_id, request_id, matched, decided_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (chat_id, message_id, request_id) DO UPDATE SET matched = excluded.matched, decided_at = excluded.decided_at`,
		decision.Key.ChatID, decision.Key.MessageID, decision.RequestID, decision.Matched, decision.DecidedAt.UTC())
	if err != nil {
		return fmt.Errorf("save decision: %w", err)
	}
	return nil
}

// MarkSent records that the «yes» of the request was shown.
func (r *SqliteMessageRepository) MarkSent(ctx context.Context, key domain.MessageKey, requestID string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE decision SET sent_at = ?
		WHERE chat_id = ? AND message_id = ? AND request_id = ?`, at.UTC(), key.ChatID, key.MessageID, requestID)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	return nil
}

// MarkProcessed marks the message processed; marking it again changes nothing.
func (r *SqliteMessageRepository) MarkProcessed(ctx context.Context, key domain.MessageKey, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO processed_message (chat_id, message_id, processed_at)
		VALUES (?, ?, ?) ON CONFLICT (chat_id, message_id) DO NOTHING`, key.ChatID, key.MessageID, at.UTC())
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return nil
}

// MoveCursor moves the chat's cursor forward; it never moves it back.
func (r *SqliteMessageRepository) MoveCursor(ctx context.Context, chatID int64, messageID int, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO chat_cursor (chat_id, last_message_id, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (chat_id) DO UPDATE SET last_message_id = max(last_message_id, excluded.last_message_id),
		updated_at = excluded.updated_at`, chatID, messageID, at.UTC())
	if err != nil {
		return fmt.Errorf("move cursor: %w", err)
	}
	return nil
}

// Cursor reads the chat's stored cursor and when it last moved; false when
// the chat has none yet.
func (r *SqliteMessageRepository) Cursor(ctx context.Context, chatID int64) (domain.StoredCursor, bool, error) {
	var stored domain.StoredCursor
	err := r.db.QueryRowContext(ctx, "SELECT last_message_id, updated_at FROM chat_cursor WHERE chat_id = ?",
		chatID).Scan(&stored.MessageID, &stored.At)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StoredCursor{}, false, nil
	}
	if err != nil {
		return domain.StoredCursor{}, false, fmt.Errorf("read cursor: %w", err)
	}
	return stored, true, nil
}
