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
	c.fail("config file not found: ./bikes.yaml", "copy tgscout.yaml.example there")

	assert.Contains(t, out.String(), "✓")
	assert.Contains(t, out.String(), "config")
	assert.Contains(t, out.String(), "./tgscout.yaml")
	assert.Contains(t, out.String(), "●")
	assert.Contains(t, out.String(), "running — press Ctrl+C to stop")
	assert.Contains(t, out.String(), "→")
	assert.Contains(t, out.String(), "\x1b[", "colored")
	assert.Contains(t, errOut.String(), "✗")
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
