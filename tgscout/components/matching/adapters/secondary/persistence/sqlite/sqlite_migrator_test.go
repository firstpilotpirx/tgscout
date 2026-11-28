package sqlite_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
)

func TestMigrationsHoldUpAndDownOfEveryStep(t *testing.T) {
	names, err := fs.Glob(sqlite.Migrations, "migrations/*.sql")

	require.NoError(t, err)
	assert.Equal(t, []string{"migrations/000001_init.down.sql", "migrations/000001_init.up.sql",
		"migrations/000002_known_peer.down.sql", "migrations/000002_known_peer.up.sql"}, names)
}
