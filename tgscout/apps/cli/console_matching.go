package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

const (
	// textStart is how much of a message or an answer a line shows.
	textStart = 80
	// matchIndent puts the text start under the time of the match line.
	matchIndent  = "         "
	unprocessed  = "the message stays unprocessed"
	llmDownHint  = unprocessed + "; is LM Studio running?"
	matchTimeFmt = "15:04"
)

// PrintMatch shows a match: the time, the request, the chat and the link,
// then the start of the text on its own line.
func (c *console) PrintMatch(match domain.Match) error {
	message := match.Message
	head := fmt.Sprintf("%s match  %s  %s  %s  %s", c.green.Sprint(c.marks.ok),
		message.SentAt.Local().Format(matchTimeFmt), match.Request.ID,
		c.cyan.Sprint(message.Chat.String()), message.Link)
	return c.block(c.out, head, matchIndent+cut(message.Text))
}

// Unclear shows an answer that was neither yes nor no, saved as «no».
func (c *console) Unclear(item domain.QueueItem, unclear application.Unclear) {
	head := fmt.Sprintf("%s unclear LLM answer for %s on %s, saved as no",
		c.yellow.Sprint(c.marks.warn), unclear.Request.ID, where(item.Message))
	_ = c.block(c.errOut, head, "  "+c.dim.Sprintf("answer: %q", cut(unclear.Raw))) // a lost line cannot be reported
}

// Failed shows a judgment that failed; the message stays unprocessed.
// It never shows the request text or the message text.
func (c *console) Failed(item domain.QueueItem, err error) {
	stage, cause, hint := "matching", err, unprocessed
	var matchErr *application.MatchError
	if errors.As(err, &matchErr) {
		stage, cause = matchErr.Stage, matchErr.Err
		if stage == application.StageLlm {
			hint = llmDownHint
		}
	}
	c.failBlock(fmt.Sprintf("%s failed on %s: %v", stage, where(item.Message), cause), hint)
}

// collectFailed shows a message the collector could not check.
func (c *console) collectFailed(message domain.Message, err error) {
	c.failBlock(fmt.Sprintf("cannot check %s: %v", where(message), err), unprocessed)
}

func (c *console) failBlock(text, hint string) {
	_ = c.block(c.errOut, c.red.Sprint(c.marks.fail)+" "+text, "  "+c.dim.Sprint(hint)) // a lost line cannot be reported
}

// where names a message by its chat, as the config writes it, and its ID.
func where(message domain.Message) string {
	return fmt.Sprintf("%s/%d", message.Chat.String(), message.Key.MessageID)
}

// cut gives the start of a text on one line: line breaks become spaces, and
// a text longer than textStart characters ends with «…».
func cut(text string) string {
	runes := []rune(strings.Join(strings.Fields(text), " "))
	if len(runes) <= textStart {
		return string(runes)
	}
	return string(runes[:textStart]) + "…"
}
