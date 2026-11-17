// Package matching wires the one component of tgscout: its use cases and the
// adapters behind their ports.
package matching

import (
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
}

// MatchingConfig holds the wired use cases of the component.
type MatchingConfig struct {
	CollectMessage *application.CollectMessage
	MatchMessage   *application.MatchMessage
}

// NewMatchingConfig wires the use cases from the given ports.
// Both use cases share one message tracker: the in-flight messages and the
// cursor of each chat.
func NewMatchingConfig(ports Ports) *MatchingConfig {
	tracker := domain.NewMessageTracker()
	return &MatchingConfig{
		CollectMessage: application.NewCollectMessage(ports.Settings, ports.Messages, ports.Queue, tracker),
		MatchMessage:   application.NewMatchMessage(ports.Messages, ports.Llm, ports.Notifier, tracker),
	}
}
