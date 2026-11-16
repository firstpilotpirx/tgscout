package application

// CollectMessage is the use case that skips a message seen before and puts a
// new one into the queue. Its behavior comes with the live-matching slice.
type CollectMessage struct {
	settings SettingsRepository
	messages MessageRepository
	queue    MessageQueue
}

// NewCollectMessage builds the use case from its ports.
func NewCollectMessage(settings SettingsRepository, messages MessageRepository, queue MessageQueue) *CollectMessage {
	return &CollectMessage{settings: settings, messages: messages, queue: queue}
}
