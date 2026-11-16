//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stopped is a context that is already cancelled: tgscout stops right after it starts.
func stopped() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestMissingConfigGivesCode1AndNoDatabase(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "bikes.yaml")

	code, _, errOut := runTgscout(stopped(), "--config", config)

	assert.Equal(t, 1, code)
	assert.Contains(t, errOut, "[fail] config file not found: "+config)
	assert.Contains(t, errOut, "copy tgscout.yaml.example there, or pass --config <path>")
	assert.NoFileExists(t, filepath.Join(dir, "tgscout.db"))
}

func TestBrokenYamlGivesCode1AndTheLine(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "tgscout.yaml")
	require.NoError(t, os.WriteFile(config, []byte("requests:\n  - id: bike\n    find: x\n    chats: a: b\n"), 0o600))

	code, _, errOut := runTgscout(stopped(), "--config", config)

	assert.Equal(t, 1, code)
	assert.Contains(t, errOut, "[fail] config is not valid YAML: "+config)
	assert.Contains(t, errOut, "line 4: mapping values are not allowed in this context")
}

func TestFirstStartCreatesTheDatabaseAndStops(t *testing.T) {
	dir := t.TempDir()
	config := writeValidConfig(t, dir)

	code, out, errOut := runTgscout(stopped(), "--config", config)

	assert.Equal(t, 0, code, errOut)
	assert.FileExists(t, filepath.Join(dir, "tgscout.db"))
	assert.Contains(t, out, "[ok] config   "+config)
	assert.Contains(t, out, "[ok] database "+filepath.Join(dir, "tgscout.db")+" created · migration 1 · 3 tables")
	assert.Contains(t, out, "[run] running — press Ctrl+C to stop")
	assert.Contains(t, out, "-> stopping…")
	assert.Contains(t, out, "[ok] stopped")
}

func TestSecondStartSaysSchemaUpToDate(t *testing.T) {
	dir := t.TempDir()
	config := writeValidConfig(t, dir)
	db := filepath.Join(dir, "other.db")
	first, _, _ := runTgscout(stopped(), "--config", config, "--db", db)
	require.Equal(t, 0, first)

	code, out, _ := runTgscout(stopped(), "--config", config, "--db", db)

	assert.Equal(t, 0, code)
	assert.Contains(t, out, "[ok] database "+db+" schema up to date · migration 1")
}

func TestDatabaseInMissingFolderGivesCode1(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "missing", "tgscout.db")

	code, _, errOut := runTgscout(stopped(), "--config", writeValidConfig(t, dir), "--db", db)

	assert.Equal(t, 1, code)
	assert.Contains(t, errOut, "[fail] database "+db)
}
