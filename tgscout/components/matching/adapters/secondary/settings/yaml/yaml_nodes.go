package yaml

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml/ast"
)

// secretKeys never belong in tgscout.yaml: they live in .env and the session file.
var secretKeys = map[string]bool{"api_id": true, "api_hash": true, "session": true}

// walker walks the YAML tree of tgscout.yaml and collects every problem.
type walker struct{ errs []FieldError }

func (w *walker) fail(line int, field, reason string) {
	w.errs = append(w.errs, FieldError{Line: line, Field: field, Reason: reason})
}

func lineOf(node ast.Node) int {
	if node == nil || node.GetToken() == nil {
		return 1
	}
	return node.GetToken().Position.Line
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// entries gives the key-value pairs of a mapping node.
func entries(node ast.Node) ([]*ast.MappingValueNode, bool) {
	switch typed := node.(type) {
	case *ast.MappingNode:
		return typed.Values, true
	case *ast.MappingValueNode:
		return []*ast.MappingValueNode{typed}, true
	}
	return nil, false
}

// fields gives the allowed keys of a mapping and reports every unknown or
// secret key; ok is false when node is not a mapping.
func (w *walker) fields(node ast.Node, path string, allowed ...string) (map[string]*ast.MappingValueNode, bool) {
	values, ok := entries(node)
	if !ok {
		w.fail(lineOf(node), path, "must be a mapping")
		return nil, false
	}
	known := map[string]bool{}
	for _, key := range allowed {
		known[key] = true
	}
	found := map[string]*ast.MappingValueNode{}
	for _, value := range values {
		key := value.Key.String()
		switch {
		case secretKeys[key]:
			w.fail(lineOf(value.Key), join(path, key), "secrets go to .env, not to tgscout.yaml")
		case !known[key]:
			w.fail(lineOf(value.Key), join(path, key), "unknown key")
			w.secretsIn(value.Value, join(path, key))
		default:
			found[key] = value
		}
	}
	return found, true
}

// secretsIn reports secret keys anywhere below an unknown key.
func (w *walker) secretsIn(node ast.Node, path string) {
	if values, ok := entries(node); ok {
		for _, value := range values {
			key := value.Key.String()
			if secretKeys[key] {
				w.fail(lineOf(value.Key), join(path, key), "secrets go to .env, not to tgscout.yaml")
			}
			w.secretsIn(value.Value, join(path, key))
		}
		return
	}
	if list, ok := node.(*ast.SequenceNode); ok {
		for i, item := range list.Values {
			w.secretsIn(item, fmt.Sprintf("%s[%d]", path, i))
		}
	}
}

func isNull(node ast.Node) bool {
	_, null := node.(*ast.NullNode)
	return node == nil || null
}

// text gives a string value; ok is false (and the problem reported) otherwise.
func (w *walker) text(node ast.Node, path string) (string, bool) {
	switch typed := node.(type) {
	case *ast.StringNode:
		return typed.Value, true
	case *ast.LiteralNode:
		return strings.TrimSpace(typed.Value.Value), true
	}
	w.fail(lineOf(node), path, "must be text")
	return "", false
}

// reference gives a chat or owner reference: text or a numeric ID.
func (w *walker) reference(node ast.Node, path string) (string, bool) {
	if number, ok := node.(*ast.IntegerNode); ok {
		return number.GetToken().Value, true
	}
	return w.text(node, path)
}

// number gives a whole number of at least min.
func (w *walker) number(node ast.Node, path string, minimum int) (int, bool) {
	value, ok := node.(*ast.IntegerNode)
	if !ok {
		w.fail(lineOf(node), path, "must be a whole number")
		return 0, false
	}
	parsed, err := strconv.Atoi(value.GetToken().Value)
	if err != nil || parsed < minimum {
		w.fail(lineOf(node), path, fmt.Sprintf("must be at least %d", minimum))
		return 0, false
	}
	return parsed, true
}

// duration gives a positive duration such as 60s.
func (w *walker) duration(node ast.Node, path string) (time.Duration, bool) {
	value, ok := node.(*ast.StringNode)
	if ok {
		parsed, err := time.ParseDuration(value.Value)
		if err == nil && parsed > 0 {
			return parsed, true
		}
	}
	w.fail(lineOf(node), path, "must be a positive duration such as 60s")
	return 0, false
}
