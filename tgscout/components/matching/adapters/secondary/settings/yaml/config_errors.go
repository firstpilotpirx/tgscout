package yaml

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrConfigNotFound means there is no config file at the given path.
var ErrConfigNotFound = errors.New("config file not found")

// ConfigSyntaxError is a config file that is not valid YAML.
type ConfigSyntaxError struct {
	Path   string
	Line   int
	Reason string
}

func (e *ConfigSyntaxError) Error() string {
	if e.Line == 0 {
		return fmt.Sprintf("config is not valid YAML: %s: %s", e.Path, e.Reason)
	}
	return fmt.Sprintf("config is not valid YAML: %s: line %d: %s", e.Path, e.Line, e.Reason)
}

// FieldError is one problem of a config field: its line, its path
// (requests[1].chats[0]) and why it is wrong.
type FieldError struct {
	Line   int
	Field  string
	Reason string
}

// ConfigErrors lists every field problem of one config file, sorted by line.
type ConfigErrors struct {
	Path   string
	Errors []FieldError
}

func (e *ConfigErrors) Error() string {
	lines := make([]string, 0, len(e.Errors))
	for _, each := range e.Errors {
		lines = append(lines, fmt.Sprintf("line %d: %s: %s", each.Line, each.Field, each.Reason))
	}
	return fmt.Sprintf("config %s has %d errors: %s", e.Path, len(e.Errors), strings.Join(lines, "; "))
}

func sortFieldErrors(errs []FieldError) {
	sort.SliceStable(errs, func(i, j int) bool {
		if errs[i].Line != errs[j].Line {
			return errs[i].Line < errs[j].Line
		}
		return errs[i].Field < errs[j].Field
	})
}
