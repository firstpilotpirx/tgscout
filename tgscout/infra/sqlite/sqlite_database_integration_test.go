//go:build integration

package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/infra/sqlite"
)

func TestOpenCreatesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tgscout.db")

	db, err := sqlite.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	assert.FileExists(t, path)
	var foreignKeys int
	require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
	assert.Equal(t, 1, foreignKeys)
}

func TestOpenInMissingFolderNamesThePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "tgscout.db")

	_, err := sqlite.Open(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}
