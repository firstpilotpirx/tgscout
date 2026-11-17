package domain

import "time"

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

// Settings is the resolved view of the config file: every default applied and
// every override resolved.
type Settings struct {
	Owner     Owner
	Workers   int
	QueueSize int
	LmStudio  LmStudio
	Requests  []Request
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
