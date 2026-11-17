package main

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

var catchUpStart = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

var colorCodes = regexp.MustCompile("\x1b\\[[0-9;]*m")

// uncolored is the text as it shows, with the symbols but without the color codes.
func uncolored(text string) string { return colorCodes.ReplaceAllString(text, "") }

func chatReport(t *testing.T, ref, title string, read int) collect.ChatReport {
	t.Helper()
	return collect.ChatReport{Chat: collect.Chat{Ref: chatRef(t, ref), Title: title}, Read: read,
		Window: domain.CatchUpChat{Depth: domain.CatchUp{Messages: 500, Days: 3}, Start: catchUpStart}}
}

func TestCatchUpStartedLine(t *testing.T) {
	c, out, _ := newTestConsole(false)

	c.catchUpStarted(3)

	assert.Equal(t, "→ catch-up started · 3 chats · live listening is on\n", uncolored(out.String()))
}

func TestCatchUpChatLineFromTheCursor(t *testing.T) {
	c, out, _ := newTestConsole(false)
	report := chatReport(t, "@made_up_market", "Riverside flea market", 214)
	report.Window.HasCursor, report.Window.After = true, 900

	c.catchUpChat(report)

	assert.Equal(t, "✓ catch-up: Riverside flea market · 214 messages from the cursor\n", uncolored(out.String()))
}

func TestCatchUpChatLineOfAFirstStart(t *testing.T) {
	c, out, _ := newTestConsole(false)

	c.catchUpChat(chatReport(t, "@made_up_swap", "Kids' things swap", 500))

	assert.Equal(t, "✓ catch-up: Kids' things swap · 500 messages · first start, last 3 days\n", uncolored(out.String()))
}

func TestCatchUpChatLineOfAGap(t *testing.T) {
	c, out, _ := newTestConsole(false)
	report := chatReport(t, "@made_up_bikes", "Bikes for sale", 1500)
	report.Window.HasCursor, report.Gap = true, true
	report.Window.CursorAt = catchUpStart.Add(-12*24*time.Hour - time.Hour)

	c.catchUpChat(report)

	assert.Equal(t, "⚠ catch-up: Bikes for sale · 1,500 messages · off for 12 days, read only the last 3 days\n", uncolored(out.String()))
}

func TestCatchUpChatLineNamesAChatWithNoTitleByItsReference(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.catchUpChat(chatReport(t, "-1001234567890", "", 1))

	assert.Equal(t, "[ok] catch-up: -1001234567890 · 1 message · first start, last 3 days\n", out.String())
}

func TestCatchUpChatLineOfAFailedRead(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	report := chatReport(t, "@made_up_market", "Riverside flea market", 0)
	report.Err = errors.New("made-up read error")

	c.catchUpChat(report)

	assert.Empty(t, out.String())
	assert.Equal(t, "[fail] catch-up: Riverside flea market · cannot read the history: made-up read error\n"+
		"  next start reads it again\n", errOut.String())
}

func TestCatchUpSummary(t *testing.T) {
	c, out, _ := newTestConsole(false)

	c.catchUpDone(collect.Summary{Chats: 3, Gaps: 1, Took: 4*time.Minute + 35*time.Second + 400*time.Millisecond,
		Counts: domain.CatchUpCounts{Read: 1042, Skipped: 18, Judged: 1024, Matches: 2}})

	assert.Equal(t,
		"✓ catch-up done in 4m35s\n"+
			"  ✓ chats      3  (1 with a gap)\n"+
			"  ✓ messages   1,042 read · 18 skipped · 1,024 judged\n"+
			"  ● matches    2  (sent to you in Telegram)\n"+
			"  ✓ errors     0\n",
		uncolored(out.String()))
}

func TestCatchUpSummaryWithErrors(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.catchUpDone(collect.Summary{Chats: 1, Took: 2 * time.Second,
		Counts: domain.CatchUpCounts{Read: 3, Judged: 3, Errors: 3}})

	assert.Equal(t,
		"[ok] catch-up done in 2s\n"+
			"  [ok] chats      1\n"+
			"  [ok] messages   3 read · 0 skipped · 3 judged\n"+
			"  [run] matches    0  (sent to you in Telegram)\n"+
			"  [fail] errors     3  (not marked processed; judged again at next start)\n",
		out.String())
}
