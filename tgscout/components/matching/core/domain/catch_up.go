package domain

import "time"

// CatchUpWindow gives how far back the chat's history is read: the larger
// number of messages and the larger number of days over its requests.
func (s Settings) CatchUpWindow(ref ChatRef) CatchUp {
	var window CatchUp
	for _, request := range s.Requests {
		for _, chat := range request.Chats {
			if chat.Ref.Key() != ref.Key() {
				continue
			}
			window.Messages = max(window.Messages, chat.CatchUp.Messages)
			window.Days = max(window.Days, chat.CatchUp.Days)
		}
	}
	return window
}

// CoveringRequests gives the requests of the chat whose own depth covers a
// history message: rank 1 is the newest message read, and the days count
// back from start.
func (s Settings) CoveringRequests(ref ChatRef, rank int, sentAt, start time.Time) []Request {
	var requests []Request
	for _, request := range s.Requests {
		for _, chat := range request.Chats {
			if chat.Ref.Key() == ref.Key() && chat.CatchUp.Covers(rank, sentAt, start) {
				requests = append(requests, request)
				break
			}
		}
	}
	return requests
}

// Covers tells whether a message of the rank, sent at sentAt, lies within
// the depth counted back from start.
func (d CatchUp) Covers(rank int, sentAt, start time.Time) bool {
	return rank <= d.Messages && !sentAt.Before(d.Since(start))
}

// Since is the oldest time the depth reaches, counted back from start.
func (d CatchUp) Since(start time.Time) time.Time {
	return start.Add(-time.Duration(d.Days) * 24 * time.Hour)
}
