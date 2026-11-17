package domain

import "time"

// MessageKey is the identity of a Telegram message: the bare peer ID of its
// chat and its message ID. Two copies of one message have the same key.
type MessageKey struct {
	ChatID    int64
	MessageID int
}

// Message is a text message of a watched chat as the collector hands it in.
// Chat is the reference as the config writes it; ChatTitle is the title
// Telegram gives the chat, empty when it gives none.
type Message struct {
	Key       MessageKey
	Chat      ChatRef
	ChatTitle string
	SentAt    time.Time
	Text      string
	Link      string
	Edited    bool
	// Live is a new message of the live listener, not an edit and not a
	// history read: only its latency counts.
	Live bool
}

// QueueItem is a message with the requests it is judged against, so the
// matcher never reads the config.
type QueueItem struct {
	Message  Message
	Requests []Request
}

// Decision is the judgment of one message against one request.
type Decision struct {
	Key       MessageKey
	RequestID string
	Matched   bool
	DecidedAt time.Time
}

// Judgment is what the LLM answered: a clear yes or no, or an unclear answer
// with its raw text.
type Judgment struct {
	Matched bool
	Unclear bool
	Raw     string
}

// Match is a message that matches a request: what the owner is told about.
type Match struct {
	Message Message
	Request Request
}
