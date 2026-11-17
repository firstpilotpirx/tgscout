package telegram

import (
	"context"
	"slices"
	"time"
)

// historyPage is how many messages one history request asks for.
const historyPage = 100

// HistoryWindow is how far back a history read goes: messages after After,
// at most Limit of them, none older than Since.
type HistoryWindow struct {
	After int
	Limit int
	Since time.Time
}

// History is what a read found: the messages, oldest first. Gap is true when
// the window stopped before it reached the message after After.
type History struct {
	Messages []Update
	Gap      bool
}

// cutWindow keeps the messages of the window from a list newest first.
func cutWindow(newestFirst []Update, window HistoryWindow) History {
	var history History
	for _, update := range newestFirst {
		if update.MessageID <= window.After {
			break
		}
		if len(history.Messages) == window.Limit || update.Date.Before(window.Since) {
			history.Gap = true
			break
		}
		history.Messages = append(history.Messages, update)
	}
	slices.Reverse(history.Messages)
	return history
}

// pageHistory fetches pages newest first, each from below the lowest ID of
// the page before, until a page is empty or the window is full, then cuts it.
func pageHistory(ctx context.Context, fetch func(ctx context.Context, offsetID int) ([]Update, error),
	window HistoryWindow) (History, error) {
	var all []Update
	offset := 0
	for {
		page, err := fetch(ctx, offset)
		if err != nil {
			return History{}, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		oldest := page[len(page)-1]
		if len(all) > window.Limit || oldest.Date.Before(window.Since) || oldest.MessageID <= window.After+1 {
			break
		}
		offset = oldest.MessageID
	}
	return cutWindow(all, window), nil
}
