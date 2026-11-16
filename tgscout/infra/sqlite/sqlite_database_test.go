package sqlite_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/infra/sqlite"
)

func TestDSNTurnsOnForeignKeysAndWal(t *testing.T) {
	assert.Equal(t, "file:/tmp/tg%20scout.db?_foreign_keys=on&_journal_mode=WAL", sqlite.DSN("/tmp/tg scout.db"))
}
