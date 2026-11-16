package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
func writeValidConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "tgscout.yaml")
	require.NoError(t, os.WriteFile(path, []byte("requests:\n  - id: bike\n    find: a kids bike\n"), 0o600))
	return path
}

func stopped() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestDefaultDatabaseSitsNextToTheConfig(t *testing.T) {
	assert.Equal(t, "./tgscout.db", databasePath("./tgscout.yaml", ""))
	assert.Equal(t, "/etc/tgscout/tgscout.db", databasePath("/etc/tgscout/tgscout.yaml", ""))
	assert.Equal(t, "/tmp/x.db", databasePath("./tgscout.yaml", "/tmp/x.db"))
}
