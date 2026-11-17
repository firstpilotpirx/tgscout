package main

import (
	"fmt"
	"strconv"
	"time"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
)

// The console lines of the catch-up at start (design D5).

const day = 24 * time.Hour

func (c *console) catchUpStarted(chats int) {
	c.step(fmt.Sprintf("catch-up started · %s · live listening is on", count(chats, "chat")))
}

// catchUpChat prints the line of one chat once its history is read.
func (c *console) catchUpChat(report collect.ChatReport) {
	name := report.Chat.Title
	if name == "" {
		name = report.Chat.Ref.String()
	}
	if report.Err != nil {
		c.failBlock(fmt.Sprintf("catch-up: %s · cannot read the history: %v", name, report.Err), "next start reads it again")
		return
	}
	read := countThousands(report.Read, "message")
	days := report.Window.Depth.Days
	switch {
	case report.Gap:
		off := int(report.Window.Start.Sub(report.Window.CursorAt) / day)
		c.println(c.out, fmt.Sprintf("%s catch-up: %s · %s · off for %s, read only the last %s",
			c.yellow.Sprint(c.marks.warn), name, read, count(off, "day"), count(days, "day")))
	case report.HasCursor():
		c.done(fmt.Sprintf("catch-up: %s · %s from the cursor", name, read))
	default:
		c.done(fmt.Sprintf("catch-up: %s · %s · first start, last %s", name, read, count(days, "day")))
	}
}

// catchUpDone prints the summary once every caught-up message is settled.
func (c *console) catchUpDone(summary collect.Summary) {
	ok, counts := c.green.Sprint(c.marks.ok), summary.Counts
	chats := strconv.Itoa(summary.Chats)
	if summary.Gaps > 0 {
		chats += fmt.Sprintf("  (%d with a gap)", summary.Gaps)
	}
	errorsLine := fmt.Sprintf("  %s errors     0", ok)
	if counts.Errors > 0 {
		errorsLine = fmt.Sprintf("  %s errors     %s  (not marked processed; judged again at next start)",
			c.red.Sprint(c.marks.fail), thousands(counts.Errors))
	}
	_ = c.block(c.out, // a lost line cannot be reported
		fmt.Sprintf("%s catch-up done in %s", ok, summary.Took.Round(time.Second)),
		fmt.Sprintf("  %s chats      %s", ok, chats),
		fmt.Sprintf("  %s messages   %s read · %s skipped · %s judged", ok,
			thousands(counts.Read), thousands(counts.Skipped), thousands(counts.Judged)),
		fmt.Sprintf("  %s matches    %s  (sent to you in Telegram)", c.cyan.Sprint(c.marks.run), thousands(counts.Matches)),
		errorsLine)
}

// thousands writes a number with commas between the thousands: 1,042.
func thousands(n int) string {
	text := strconv.Itoa(n)
	for at := len(text) - 3; at > 0 && text[at-1] != '-'; at -= 3 {
		text = text[:at] + "," + text[at:]
	}
	return text
}

func countThousands(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return thousands(n) + " " + noun + "s"
}
