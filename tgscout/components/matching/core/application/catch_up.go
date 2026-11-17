package application

import (
	"context"
	"sync"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// CatchUp is the use case that reads where each chat's history starts and
// how deep it goes, hands each history message on with the requests whose
// depth covers it, and holds the chat's cursor until its history is handed on.
type CatchUp struct {
	settings SettingsRepository
	messages MessageRepository
	collect  *CollectMessage
	tracker  *domain.MessageTracker
	now      func() time.Time
	once     sync.Once
	start    time.Time
}

// NewCatchUp builds the use case; now gives the time catch-up begins.
func NewCatchUp(settings SettingsRepository, messages MessageRepository, collect *CollectMessage,
	tracker *domain.MessageTracker, now func() time.Time) *CatchUp {
	return &CatchUp{settings: settings, messages: messages, collect: collect, tracker: tracker, now: now}
}

// Open holds the chat's cursor and gives where its read starts and how deep
// it goes; the depth counts from the first Open of the run.
func (c *CatchUp) Open(ctx context.Context, chatID int64, ref domain.ChatRef) (domain.CatchUpChat, error) {
	c.once.Do(func() { c.start = c.now() })
	c.tracker.Hold(chatID)
	chat := domain.CatchUpChat{ChatID: chatID, Ref: ref, Depth: c.settings.Settings().CatchUpWindow(ref), Start: c.start}
	stored, found, err := c.messages.Cursor(ctx, chatID)
	if err != nil {
		c.tracker.Release(chatID)
		return chat, err
	}
	if found {
		chat.After, chat.HasCursor, chat.CursorAt = stored.MessageID, true, stored.At
	}
	return chat, nil
}

// Collect hands one history message on and says whether it went to the
// judge; rank 1 is the newest message read.
func (c *CatchUp) Collect(ctx context.Context, chat domain.CatchUpChat, message domain.Message, rank int) (bool, error) {
	requests := c.settings.Settings().CoveringRequests(chat.Ref, rank, message.SentAt, chat.Start)
	return c.collect.CollectHistory(ctx, message, requests)
}

// Pause holds the cursor of each chat while the connection to Telegram is
// away; the gap read after it opens and closes each chat again.
func (c *CatchUp) Pause(chatIDs []int64) {
	for _, chatID := range chatIDs {
		c.tracker.Hold(chatID)
	}
}

// Close releases the chat's cursor and stores it when it moved.
func (c *CatchUp) Close(ctx context.Context, chat domain.CatchUpChat) error {
	cursor, moved := c.tracker.Release(chat.ChatID)
	if !moved {
		return nil
	}
	return c.messages.MoveCursor(ctx, chat.ChatID, cursor, time.Now())
}

// Wait gives the counts once the judge settled every history message.
func (c *CatchUp) Wait(ctx context.Context) (domain.CatchUpCounts, error) {
	return c.tracker.Tally().Wait(ctx)
}
