//go:build integration

package telegram

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorageLoadsAnExistingSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tgscout.session")
	require.NoError(t, os.WriteFile(path, []byte("made-up session"), 0o600))
	storage := NewFileStorage(path, false)

	data, err := storage.LoadSession(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "made-up session", string(data))
}

func TestFileStorageWritesNothingBeforeCommit(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tgscout.session")
	storage := NewFileStorage(path, true)

	_, err := storage.LoadSession(ctx)
	require.True(t, errors.Is(err, session.ErrNotFound))
	require.NoError(t, storage.StoreSession(ctx, []byte("made-up key")))
	data, err := storage.LoadSession(ctx)
	require.NoError(t, err)
	assert.Equal(t, "made-up key", string(data), "gotd reads back what it stored")
	assert.NoFileExists(t, path)

	require.NoError(t, storage.Commit(ctx))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	saved, err := os.ReadFile(path) //nolint:gosec // G304: the test's own temp file
	require.NoError(t, err)
	assert.Equal(t, "made-up key", string(saved))
}

func TestFileStorageWritesThroughAfterCommit(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tgscout.session")
	storage := NewFileStorage(path, true)
	require.NoError(t, storage.Commit(ctx))
	assert.NoFileExists(t, path, "nothing stored, nothing written")

	require.NoError(t, storage.StoreSession(ctx, []byte("later key")))

	saved, err := os.ReadFile(path) //nolint:gosec // G304: the test's own temp file
	require.NoError(t, err)
	assert.Equal(t, "later key", string(saved))
}
