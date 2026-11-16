package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestConsole(plain bool) (*console, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return newConsole(&out, &errOut, plain), &out, &errOut
}

func TestConsoleMarksWithSymbols(t *testing.T) {
	c, out, errOut := newTestConsole(false)

	c.ok("config", "./tgscout.yaml", "")
	c.running()
	c.step("stopping…")
	c.done("stopped")
	c.fail("config file not found: ./bikes.yaml", "copy tgscout.yaml.example there", "./bikes.yaml")

	assert.Contains(t, out.String(), "✓")
	assert.Contains(t, out.String(), "config")
	assert.Contains(t, out.String(), "./tgscout.yaml")
	assert.Contains(t, out.String(), "●")
	assert.Contains(t, out.String(), "running — press Ctrl+C to stop")
	assert.Contains(t, out.String(), "→")
	assert.Contains(t, out.String(), "\x1b[", "colored")
	assert.Contains(t, errOut.String(), "✗")
	assert.Contains(t, errOut.String(), "\x1b[31;1m./bikes.yaml", "the offending path is red")
	assert.Contains(t, out.String(), "\x1b[36m./tgscout.yaml", "paths are cyan")
	assert.Contains(t, errOut.String(), "\n  ")
	assert.Contains(t, errOut.String(), "copy tgscout.yaml.example there")
}

func TestConsolePlainUsesTextMarksAndNoColor(t *testing.T) {
	c, out, errOut := newTestConsole(true)

	c.ok("database", "./tgscout.db", "created · migration 1 · 3 tables")
	c.running()
	c.step("stopping…")
	c.done("stopped")
	c.fail("config is not valid YAML: ./tgscout.yaml", "line 4: bad")

	assert.Equal(t,
		"[ok] database ./tgscout.db created · migration 1 · 3 tables\n"+
			"[run] running — press Ctrl+C to stop\n"+
			"-> stopping…\n"+
			"[ok] stopped\n",
		out.String())
	assert.Equal(t, "[fail] config is not valid YAML: ./tgscout.yaml\n  line 4: bad\n", errOut.String())
}

func TestPlainOutputFollowsNoColorAndForceColor(t *testing.T) {
	assert.True(t, plainOutput(true, ""))
	assert.False(t, plainOutput(false, ""))
	assert.False(t, plainOutput(true, "1"))
}

func TestConsoleWarnUsesWarnMark(t *testing.T) {
	c, _, errOut := newTestConsole(false)
	plain, _, plainErr := newTestConsole(true)

	c.warn("./tgscout.yaml", "101 chats; the design maximum is 100")
	plain.warn("./tgscout.yaml", "101 chats; the design maximum is 100")

	assert.Contains(t, errOut.String(), "⚠")
	assert.Contains(t, errOut.String(), "\x1b[", "colored")
	assert.Equal(t, "[warn] ./tgscout.yaml: 101 chats; the design maximum is 100\n", plainErr.String())
}

func TestConsoleSummaryNamesRequestsChatsAndModel(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.summary("./tgscout.yaml", 3, 7, "example-model")
	c.summary("./one.yaml", 1, 1, "example-model")

	assert.Equal(t, "[ok] ./tgscout.yaml: 3 requests, 7 chats, model example-model\n"+
		"[ok] ./one.yaml: 1 request, 1 chat, model example-model\n", out.String())
}

func TestConsoleFailListAlignsLineAndField(t *testing.T) {
	c, _, errOut := newTestConsole(true)

	c.failList("./tgscout.yaml", []listRow{
		{line: 4, field: "requests[0].chats", reason: "a request needs at least one chat"},
		{line: 15, field: "api_hash", reason: "secrets go to .env, not to tgscout.yaml"},
	})

	assert.Equal(t, "[fail] ./tgscout.yaml: 2 errors\n"+
		"  line  4  requests[0].chats  a request needs at least one chat\n"+
		"  line 15  api_hash           secrets go to .env, not to tgscout.yaml\n", errOut.String())
}

func TestConsoleFailListOfOneSaysOneError(t *testing.T) {
	c, _, errOut := newTestConsole(true)

	c.failList("./tgscout.yaml", []listRow{{line: 1, field: "owner", reason: "is required"}})

	assert.Equal(t, "[fail] ./tgscout.yaml: 1 error\n  line 1  owner  is required\n", errOut.String())
}
