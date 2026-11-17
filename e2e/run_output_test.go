package e2e_test

import (
	"fmt"
	"strings"
)

func (w *world) allOut(stderr bool) string {
	var text strings.Builder
	for _, run := range w.matching.runs {
		if stderr {
			text.WriteString(run.stderr.String())
		} else {
			text.WriteString(run.stdout.String())
		}
	}
	return text.String()
}

func (w *world) allStdoutHas(want ...string) error { return has("stdout", w.allOut(false), want) }
func (w *world) allStderrHas(want ...string) error { return has("stderr", w.allOut(true), want) }

// matchLines checks how many match lines of a request (any, when empty) every run printed.
func (w *world) matchLines(request string, want int) error {
	got := 0
	for _, line := range strings.Split(w.allOut(false), "\n") {
		if strings.HasPrefix(line, "[ok] match sent  ") && strings.Contains(line, "  "+request) {
			got++
		}
	}
	if got != want {
		return fmt.Errorf("%d match lines of %q, want %d; stdout:\n%s", got, request, want, w.allOut(false))
	}
	return nil
}
