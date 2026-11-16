package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runTgscout(ctx context.Context, args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(ctx, args, &out, &errOut, true)
	return code, out.String(), errOut.String()
}

func TestHelpPrintsUsageWithEveryFlag(t *testing.T) {
	code, out, errOut := runTgscout(context.Background(), "--help")

	assert.Equal(t, 0, code)
	assert.Empty(t, errOut)
	assert.Contains(t, out, "tgscout watches Telegram chats and writes to you when a message matches.")
	assert.Contains(t, out, "tgscout [flags]")
	assert.Contains(t, out, `--config string   config file (default "./tgscout.yaml")`)
	assert.Contains(t, out, "--db string       SQLite file (default: tgscout.db next to the config)")
	assert.Contains(t, out, "-h, --help            show this help")
}

func TestUnknownFlagGivesUsageOnStderrAndCode2(t *testing.T) {
	dir := t.TempDir()

	code, out, errOut := runTgscout(context.Background(), "--colour", "--config", writeValidConfig(t, dir))

	assert.Equal(t, 2, code)
	assert.Empty(t, out)
	assert.Contains(t, errOut, "[fail] unknown flag: --colour")
	assert.Contains(t, errOut, "tgscout [flags]")
	assert.NoFileExists(t, dir+"/tgscout.db")
}

func TestStrayArgumentGivesCode2(t *testing.T) {
	code, _, errOut := runTgscout(context.Background(), "run")

	assert.Equal(t, 2, code)
	assert.Contains(t, errOut, `[fail] unknown argument "run"`)
	assert.Contains(t, errOut, "tgscout [flags]")
}
