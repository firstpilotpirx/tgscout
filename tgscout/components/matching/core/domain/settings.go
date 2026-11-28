package domain

import (
	"net/url"
	"strings"
	"time"
)

// CatchUp is the resolved catch-up depth of one chat of one request.
type CatchUp struct {
	Messages int
	Days     int
}

// RequestChat is one chat a request watches, with its resolved depth.
type RequestChat struct {
	Ref     ChatRef
	CatchUp CatchUp
}

// Request is a standing request: the text the LLM compares a message with and
// the chats it watches.
type Request struct {
	ID    string
	Find  string
	Chats []RequestChat
}

// LmStudio is how tgscout calls the local LLM.
type LmStudio struct {
	URL     string
	Model   string
	Timeout time.Duration
}

// Local tells whether the URL's host is this machine (localhost, 127.0.0.1
// or ::1), so chat texts stay on it.
func (l LmStudio) Local() bool {
	parsed, err := url.Parse(l.URL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// Settings is the resolved view of the config file: every default applied and
// every override resolved.
type Settings struct {
	Owner     Owner
	Workers   int
	QueueSize int
	// RetryMaxPause is the longest pause between two tries of LM Studio or a send.
	RetryMaxPause time.Duration
	// StopGrace is how long a stop waits for the messages in work.
	StopGrace time.Duration
	// StatusInterval is how often the status line is printed.
	StatusInterval time.Duration
	// HistoryPause is the least time between the starts of two history
	// calls to Telegram; 0 turns the pause off.
	HistoryPause time.Duration
	LmStudio     LmStudio
	Requests     []Request
}

// RequestsOf gives every request that watches the chat, in config order.
func (s Settings) RequestsOf(ref ChatRef) []Request {
	var requests []Request
	for _, request := range s.Requests {
		for _, chat := range request.Chats {
			if chat.Ref.Key() == ref.Key() {
				requests = append(requests, request)
				break
			}
		}
	}
	return requests
}

// DistinctChats counts the chats of all requests, each chat once.
func (s Settings) DistinctChats() int {
	seen := map[string]bool{}
	for _, request := range s.Requests {
		for _, chat := range request.Chats {
			seen[chat.Ref.Key()] = true
		}
	}
	return len(seen)
}
