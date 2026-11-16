package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validConfig is a made-up config with every required field.
const validConfig = "owner: '@made_up_owner'\nlm_studio:\n  model: example-model\n" +
	"requests:\n  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market']\n"

func writeValidConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "tgscout.yaml")
	require.NoError(t, os.WriteFile(path, []byte(validConfig), 0o600))
	return path
}

func TestDefaultDatabaseSitsNextToTheConfig(t *testing.T) {
	assert.Equal(t, "./tgscout.db", databasePath("./tgscout.yaml", ""))
	assert.Equal(t, "/etc/tgscout/tgscout.db", databasePath("/etc/tgscout/tgscout.yaml", ""))
	assert.Equal(t, "/tmp/x.db", databasePath("./tgscout.yaml", "/tmp/x.db"))
}
