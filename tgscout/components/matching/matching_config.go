// Package matching wires the one component of tgscout: its use cases and the
// adapters behind their ports.
package matching

import (
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Ports holds the driven adapters the app passes in.
type Ports struct {
	Settings application.SettingsRepository
	Messages application.MessageRepository
	Queue    application.MessageQueue
	Llm      application.Llm
	Notifier application.Notifier
	// Delivery is how judging and sending retry: the pauses, the sleep, the reports.
	Delivery application.Delivery
	// Stats counts the run for the status line; nil counts nothing.
	Stats *domain.RunStats
}

// MatchingConfig holds the wired use cases of the component.
type MatchingConfig struct {
	CollectMessage *application.CollectMessage
	MatchMessage   *application.MatchMessage
	CatchUp        *application.CatchUp
}

// NewMatchingConfig wires the use cases from the given ports.
// The use cases share one message tracker: the in-flight messages, the
// cursor of each chat and the catch-up tally.
func NewMatchingConfig(ports Ports) *MatchingConfig {
	tracker := domain.NewMessageTracker()
	collect := application.NewCollectMessage(ports.Settings, ports.Messages, ports.Queue, tracker)
	collect.Counting(ports.Stats)
	match := application.NewMatchMessage(ports.Messages, ports.Llm, ports.Notifier, tracker, ports.Delivery)
	match.Counting(ports.Stats)
	return &MatchingConfig{
		CollectMessage: collect,
		MatchMessage:   match,
		CatchUp:        application.NewCatchUp(ports.Settings, ports.Messages, collect, tracker, time.Now),
	}
}
