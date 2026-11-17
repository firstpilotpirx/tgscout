//go:build integration

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
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
	assert.Contains(t, errOut, "line 4: mapping value is not allowed in this context")
}

func TestFirstStartCreatesTheDatabaseAndStops(t *testing.T) {
	dir := t.TempDir()
	config := writeValidConfig(t, dir)

	code, out, errOut := runTgscout(stopped(), "--config", config)

	assert.Equal(t, 0, code, errOut)
	assert.FileExists(t, filepath.Join(dir, "tgscout.db"))
	assert.Contains(t, out, "[ok] "+config+": 1 request, 1 chat, model example-model")
	assert.Contains(t, out, "[ok] database "+filepath.Join(dir, "tgscout.db")+" created · migration 1 · 3 tables")
	assert.Contains(t, out, "[run] running — press Ctrl+C to stop")
	assert.Contains(t, out, "-> stopping · finishing 0 messages in work · up to 10s · Ctrl+C again to stop now")
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

func TestInvalidConfigGivesCode1EveryErrorAndNoDatabase(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "tgscout.yaml")
	require.NoError(t, os.WriteFile(config, []byte("owner: '@made_up_owner'\nlm_studio: {model: m}\n"+
		"requests:\n  - id: bike\n    find: x\n    chats: []\napi_hash: x\n"), 0o600))

	code, out, errOut := runTgscout(stopped(), "--config", config)

	assert.Equal(t, 1, code)
	assert.NotContains(t, out, "[ok]")
	assert.Contains(t, errOut, "[fail] "+config+": 2 errors")
	assert.Contains(t, errOut, "line 6  requests[0].chats  a request needs at least one chat")
	assert.Contains(t, errOut, "line 7  api_hash           secrets go to .env, not to tgscout.yaml")
	assert.NoFileExists(t, filepath.Join(dir, "tgscout.db"))
}

func TestTooManyChatsWarnsAndStarts(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "tgscout.yaml")
	chats := make([]string, 0, 101)
	for i := range 101 {
		chats = append(chats, fmt.Sprintf("'%d'", 1000+i))
	}
	require.NoError(t, os.WriteFile(config, []byte("owner: '@made_up_owner'\nlm_studio: {model: m}\n"+
		"requests:\n  - id: bike\n    find: x\n    chats: ["+strings.Join(chats, ", ")+"]\n"), 0o600))

	code, out, errOut := runTgscout(stopped(), "--config", config)

	assert.Equal(t, 0, code, errOut)
	assert.Contains(t, errOut, "[warn] "+config+": 101 chats; the design maximum is 100")
	assert.Contains(t, out, "[ok] "+config+": 1 request, 101 chats, model m")
}

func TestLookupFailureGivesCode1(t *testing.T) {
	dir := t.TempDir()
	deps := testDeps()
	deps.newClient = func(telegram.Config) (telegram.Client, error) {
		client := everythingFound()
		client.authorized = true
		client.chats["@made_up_market"] = telegram.ChatNotJoined
		return client, nil
	}
	var out, errOut bytes.Buffer

	code := runWith(stopped(), []string{"--config", writeValidConfig(t, dir)}, newConsole(&out, &errOut, true), deps)

	assert.Equal(t, 1, code)
	assert.Contains(t, errOut.String(), "@made_up_market  the account has not joined it")
	assert.NotContains(t, out.String(), "running")
}
