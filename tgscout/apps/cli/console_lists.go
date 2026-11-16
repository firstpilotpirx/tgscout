package main

import (
	"fmt"
	"strings"
)

// listRow is one line of an error list: where the problem is and why.
type listRow struct {
	line   int
	field  string
	reason string
}

// summary prints the line of a valid config: requests, chats and model.
func (c *console) summary(path string, requests, chats int, model string) {
	c.println(c.out, fmt.Sprintf("%s %s: %s, %s, model %s", c.green.Sprint(c.marks.ok), c.cyan.Sprint(path),
		count(requests, "request"), count(chats, "chat"), model))
}

// warn prints a problem that does not stop the start to errOut.
func (c *console) warn(path, text string) {
	c.println(c.errOut, fmt.Sprintf("%s %s: %s", c.yellow.Sprint(c.marks.warn), c.cyan.Sprint(path), text))
}

// failList prints every problem of a file to errOut: a title with their
// count, then one aligned line each.
func (c *console) failList(path string, rows []listRow) {
	c.println(c.errOut, fmt.Sprintf("%s %s: %s", c.red.Sprint(c.marks.fail), c.red.Sprint(path), count(len(rows), "error")))
	lineWidth, fieldWidth := 0, 0
	for _, row := range rows {
		lineWidth = max(lineWidth, len(fmt.Sprint(row.line)))
		fieldWidth = max(fieldWidth, len(row.field))
	}
	for _, row := range rows {
		line := c.dim.Sprintf("line %*d", lineWidth, row.line)
		field := row.field + strings.Repeat(" ", fieldWidth-len(row.field))
		c.println(c.errOut, fmt.Sprintf("  %s  %s  %s", line, c.cyan.Sprint(field), row.reason))
	}
}

// count writes a number with its noun: "1 chat", "7 chats".
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
