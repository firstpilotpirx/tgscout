package yaml

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	goyaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Defaults of tgscout.yaml.
const (
	DefaultWorkers = 2
	// DefaultQueueSize is how many messages may wait for the LLM.
	DefaultQueueSize = 1000
	DefaultURL       = "http://localhost:1234/v1"
	DefaultTimeout   = 60 * time.Second
	// DefaultRetryMaxPause is the longest pause between two tries.
	DefaultRetryMaxPause = 60 * time.Second
	// DefaultStopGrace is how long a stop waits for the messages in work.
	DefaultStopGrace = 10 * time.Second
	// DefaultStatusInterval is how often the status line is printed.
	DefaultStatusInterval = 5 * time.Minute
	// MinStatusInterval is the shortest status interval.
	MinStatusInterval = 10 * time.Second
	// DefaultHistoryPause is the least time between two history calls.
	DefaultHistoryPause = time.Second
	// MaxChats is the design maximum of distinct chats; more gives a warning.
	MaxChats = 100
)

// Parse reads the text of tgscout.yaml into resolved settings. It returns
// the warnings, a *ConfigSyntaxError for text that is not YAML, or a
// *ConfigErrors with every field problem at once.
func Parse(data []byte) (domain.Settings, []string, error) {
	file, err := parser.ParseBytes(data, 0)
	if err != nil {
		return domain.Settings{}, nil, syntaxError(err)
	}
	var body ast.Node
	if len(file.Docs) > 0 {
		body = file.Docs[0].Body
	}
	w := &walker{}
	settings := w.root(body)
	if len(w.errs) > 0 {
		sortFieldErrors(w.errs)
		return domain.Settings{}, nil, &ConfigErrors{Errors: w.errs}
	}
	var warnings []string
	if count := settings.DistinctChats(); count > MaxChats {
		warnings = append(warnings, fmt.Sprintf("%d chats; the design maximum is %d", count, MaxChats))
	}
	return settings, warnings, nil
}

func syntaxError(err error) *ConfigSyntaxError {
	var syntax *goyaml.SyntaxError
	if errors.As(err, &syntax) && syntax.Token != nil {
		return &ConfigSyntaxError{Line: syntax.Token.Position.Line, Reason: syntax.Message}
	}
	return &ConfigSyntaxError{Reason: err.Error()}
}

func (w *walker) root(body ast.Node) domain.Settings {
	if isNull(body) {
		body = &ast.MappingNode{BaseNode: &ast.BaseNode{}}
	}
	found, ok := w.fields(body, "", "owner", "workers", "queue_size", "retry_max_pause", "stop_grace", "status_interval", "history_pause", "lm_studio", "catch_up", "requests")
	if !ok {
		return domain.Settings{}
	}
	settings := domain.Settings{
		Owner:          w.owner(found["owner"], lineOf(body)),
		Workers:        DefaultWorkers,
		QueueSize:      DefaultQueueSize,
		RetryMaxPause:  DefaultRetryMaxPause,
		StopGrace:      DefaultStopGrace,
		StatusInterval: DefaultStatusInterval,
		HistoryPause:   DefaultHistoryPause,
		LmStudio:       w.lmStudio(found["lm_studio"], lineOf(body)),
	}
	w.timing(found, &settings)
	if queueSize := found["queue_size"]; queueSize != nil {
		settings.QueueSize, _ = w.number(queueSize.Value, "queue_size", 1)
	}
	if pause := found["retry_max_pause"]; pause != nil {
		settings.RetryMaxPause, _ = w.duration(pause.Value, "retry_max_pause")
	}
	if grace := found["stop_grace"]; grace != nil {
		settings.StopGrace, _ = w.duration(grace.Value, "stop_grace")
	}
	if workers := found["workers"]; workers != nil {
		settings.Workers, _ = w.number(workers.Value, "workers", 1)
	}
	rootDepth := builtInDepth
	if catchUp := found["catch_up"]; catchUp != nil {
		rootDepth = w.depth(catchUp.Value, "catch_up").over(builtInDepth)
	}
	settings.Requests = w.requests(found["requests"], lineOf(body), rootDepth)
	return settings
}

// timing reads the root keys of the status line and the history pace.
func (w *walker) timing(found map[string]*ast.MappingValueNode, settings *domain.Settings) {
	if interval := found["status_interval"]; interval != nil {
		settings.StatusInterval = w.statusInterval(interval.Value)
	}
	if pause := found["history_pause"]; pause != nil {
		settings.HistoryPause = w.historyPause(pause.Value)
	}
}

// statusInterval reads a duration of at least MinStatusInterval.
func (w *walker) statusInterval(node ast.Node) time.Duration {
	if value, ok := node.(*ast.StringNode); ok {
		parsed, err := time.ParseDuration(value.Value)
		if err == nil && parsed >= MinStatusInterval {
			return parsed
		}
	}
	w.fail(lineOf(node), "status_interval", "must be a duration of at least 10s such as 5m")
	return DefaultStatusInterval
}

// historyPause reads the pause between two history calls: a duration of 0s
// or more, where 0s turns the pause off.
func (w *walker) historyPause(node ast.Node) time.Duration {
	if value, ok := node.(*ast.StringNode); ok {
		if parsed, err := time.ParseDuration(value.Value); err == nil && parsed >= 0 {
			return parsed
		}
	}
	w.fail(lineOf(node), "history_pause", "must be 0s or more, a duration such as 1s")
	return DefaultHistoryPause
}

func (w *walker) owner(entry *ast.MappingValueNode, parentLine int) domain.Owner {
	if entry == nil || isNull(entry.Value) {
		w.fail(parentLine, "owner", "is required: @username or a numeric ID")
		return domain.Owner{}
	}
	text, ok := w.reference(entry.Value, "owner")
	if !ok {
		return domain.Owner{}
	}
	owner, err := domain.NewOwner(text)
	if err != nil {
		w.fail(lineOf(entry.Value), "owner", err.Error())
	}
	return owner
}

func (w *walker) lmStudio(entry *ast.MappingValueNode, parentLine int) domain.LmStudio {
	settings := domain.LmStudio{URL: DefaultURL, Timeout: DefaultTimeout}
	if entry == nil {
		w.fail(parentLine, "lm_studio.model", "is required")
		return settings
	}
	found, ok := w.fields(entry.Value, "lm_studio", "url", "model", "timeout")
	if !ok {
		return settings
	}
	if model := found["model"]; model != nil && !isNull(model.Value) {
		settings.Model, _ = w.text(model.Value, "lm_studio.model")
	} else {
		w.fail(lineOf(entry.Key), "lm_studio.model", "is required")
	}
	if address := found["url"]; address != nil {
		settings.URL = w.address(address.Value)
	}
	if timeout := found["timeout"]; timeout != nil {
		settings.Timeout, _ = w.duration(timeout.Value, "lm_studio.timeout")
	}
	return settings
}

func (w *walker) address(node ast.Node) string {
	text, ok := w.text(node, "lm_studio.url")
	if !ok {
		return ""
	}
	parsed, err := url.Parse(text)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		w.fail(lineOf(node), "lm_studio.url", "must be an http or https address")
	}
	return text
}
