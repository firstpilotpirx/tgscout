//go:build integration

package sqlite_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
	sqlitedb "github.com/firstpilotpirx/tgscout/infra/sqlite"
)

func openTemp(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tgscout.db")
	db, err := sqlitedb.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, path
}

func TestMigrateNewFileGetsTheSchema(t *testing.T) {
	db, _ := openTemp(t)

	result, err := sqlite.Migrate(db)

	require.NoError(t, err)
	assert.Equal(t, sqlite.MigrationResult{Version: 2, Tables: 4, Applied: true}, result)
	for _, table := range []string{"chat_cursor", "processed_message", "decision", "known_peer"} {
		var name string
		require.NoError(t, db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name))
	}
}

func TestMigrateSecondRunKeepsRows(t *testing.T) {
	db, _ := openTemp(t)
	_, err := sqlite.Migrate(db)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO chat_cursor (chat_id, last_message_id, updated_at) VALUES (7, 42, CURRENT_TIMESTAMP)")
	require.NoError(t, err)

	result, err := sqlite.Migrate(db)

	require.NoError(t, err)
	assert.Equal(t, sqlite.MigrationResult{Version: 2, Tables: 4, Applied: false}, result)
	var last int64
	require.NoError(t, db.QueryRow("SELECT last_message_id FROM chat_cursor WHERE chat_id = 7").Scan(&last))
	assert.Equal(t, int64(42), last)
}
