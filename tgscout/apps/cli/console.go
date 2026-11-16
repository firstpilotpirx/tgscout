package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// marks are the status marks of one line; plainMarks replace the symbols
// when color is off (terminal-output.md).
type marks struct{ ok, fail, warn, run, step string }

var (
	symbolMarks = marks{ok: "✓", fail: "✗", warn: "⚠", run: "●", step: "→"}
	plainMarks  = marks{ok: "[ok]", fail: "[fail]", warn: "[warn]", run: "[run]", step: "->"}
)

// console writes the status lines of tgscout: results to out, errors to errOut.
type console struct {
	out, errOut io.Writer
	marks       marks
	green, red  *color.Color
	cyan, dim   *color.Color
	yellow      *color.Color
}

// plainOutput tells whether output goes without color: color.NoColor covers a
// non-terminal stdout, NO_COLOR and TERM=dumb; FORCE_COLOR=1 turns color back on.
func plainOutput(noColor bool, forceColor string) bool {
	if forceColor == "1" {
		return false
	}
	return noColor
}

func newConsole(out, errOut io.Writer, plain bool) *console {
	c := &console{
		out: out, errOut: errOut, marks: symbolMarks,
		green: color.New(color.FgGreen), red: color.New(color.FgRed, color.Bold),
		cyan: color.New(color.FgCyan), dim: color.New(color.Faint),
		yellow: color.New(color.FgYellow),
	}
	for _, each := range []*color.Color{c.green, c.red, c.cyan, c.dim, c.yellow} {
		if plain {
			each.DisableColor()
		} else {
			each.EnableColor()
		}
	}
	if plain {
		c.marks = plainMarks
	}
	return c
}

// ok prints a success line: the label, the path and an optional detail.
func (c *console) ok(label, path, detail string) {
	line := fmt.Sprintf("%s %-8s %s", c.green.Sprint(c.marks.ok), label, c.cyan.Sprint(path))
	if detail != "" {
		line += " " + detail
	}
	c.println(c.out, line)
}

func (c *console) running() {
	c.println(c.out, c.cyan.Sprint(c.marks.run)+" running — press Ctrl+C to stop")
}

func (c *console) step(text string) { c.println(c.out, c.cyan.Sprint(c.marks.step)+" "+text) }

func (c *console) done(text string) { c.println(c.out, c.green.Sprint(c.marks.ok)+" "+text) }

// fail prints an error line and its hint to errOut; each path in paths is
// shown red inside the text.
func (c *console) fail(text, hint string, paths ...string) {
	for _, path := range paths {
		text = strings.ReplaceAll(text, path, c.red.Sprint(path))
	}
	c.println(c.errOut, c.red.Sprint(c.marks.fail)+" "+text)
	if hint != "" {
		c.println(c.errOut, "  "+c.dim.Sprint(hint))
	}
}

func (c *console) println(w io.Writer, line string) {
	_, _ = fmt.Fprintln(w, line) // a lost console line cannot be reported anywhere
}
