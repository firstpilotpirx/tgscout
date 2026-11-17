//go:build integration

package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

var at = time.Date(2026, 1, 2, 14, 2, 0, 0, time.UTC)

func newRepository(t *testing.T) (*sqlite.SqliteMessageRepository, func(query string, args ...any) int) {
	db, _ := openTemp(t)
	_, err := sqlite.Migrate(db)
	require.NoError(t, err)
	count := func(query string, args ...any) int {
		var n int
		require.NoError(t, db.QueryRow(query, args...).Scan(&n))
		return n
	}
	return sqlite.NewSqliteMessageRepository(db), count
}

func TestRepositorySavesAndReplacesADecision(t *testing.T) {
	repository, count := newRepository(t)
	ctx := context.Background()
	key := domain.MessageKey{ChatID: 7, MessageID: 1}

	require.NoError(t, repository.SaveDecision(ctx, domain.Decision{Key: key, RequestID: "bike", Matched: false, DecidedAt: at}))
	require.NoError(t, repository.SaveDecision(ctx, domain.Decision{Key: key, RequestID: "bike", Matched: true, DecidedAt: at}))

	assert.Equal(t, 1, count("SELECT count(*) FROM decision"))
	assert.Equal(t, 1, count("SELECT count(*) FROM decision WHERE matched = 1"))
}

func TestRepositoryListsOnlyTheSentRequests(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()
	key := domain.MessageKey{ChatID: 7, MessageID: 1}
	for _, id := range []string{"bike", "lamp"} {
		require.NoError(t, repository.SaveDecision(ctx, domain.Decision{Key: key, RequestID: id, Matched: true, DecidedAt: at}))
	}

	require.NoError(t, repository.MarkSent(ctx, key, "lamp", at))
	sent, err := repository.SentRequests(ctx, key)

	require.NoError(t, err)
	assert.Equal(t, []string{"lamp"}, sent)
}

func TestRepositoryMarksAMessageProcessed(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()
	key := domain.MessageKey{ChatID: 7, MessageID: 1}

	before, err := repository.IsProcessed(ctx, key)
	require.NoError(t, err)
	require.NoError(t, repository.MarkProcessed(ctx, key, at))
	require.NoError(t, repository.MarkProcessed(ctx, key, at))
	after, err := repository.IsProcessed(ctx, key)

	require.NoError(t, err)
	assert.False(t, before)
	assert.True(t, after)
}

func TestRepositoryCursorNeverMovesBack(t *testing.T) {
	repository, count := newRepository(t)
	ctx := context.Background()

	require.NoError(t, repository.MoveCursor(ctx, 7, 12, at))
	require.NoError(t, repository.MoveCursor(ctx, 7, 10, at))

	assert.Equal(t, 12, count("SELECT last_message_id FROM chat_cursor WHERE chat_id = 7"))
}
