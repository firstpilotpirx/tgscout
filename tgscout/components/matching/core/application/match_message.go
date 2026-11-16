package application

// MatchMessage is the use case that asks the LLM once per request, saves each
// decision and sends every match to the owner. Its behavior comes with the
// live-matching slice.
type MatchMessage struct {
	messages MessageRepository
	llm      Llm
	notifier Notifier
}

// NewMatchMessage builds the use case from its ports.
func NewMatchMessage(messages MessageRepository, llm Llm, notifier Notifier) *MatchMessage {
	return &MatchMessage{messages: messages, llm: llm, notifier: notifier}
}
