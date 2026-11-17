// Package telegram turns the Telegram updates of the watched chats into
// messages for the collect use case.
package telegram

import (
	"context"
	"fmt"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// Chat is a watched chat as the start found it: its bare peer ID, the config
// reference, its username, empty for a private chat, and its title, empty
// when none.
type Chat struct {
	ID       int64
	Ref      domain.ChatRef
	Username string
	Title    string
}

// MessageCollector is the collect use case.
type MessageCollector interface {
	Collect(ctx context.Context, message domain.Message) error
}

// TelegramCollector hands each text update of a watched chat to the use case.
type TelegramCollector struct {
	chats   map[int64]Chat
	useCase MessageCollector
	report  func(domain.Message, error)
}

// NewTelegramCollector builds the collector over the found chats, by peer ID;
// report prints a collect error.
func NewTelegramCollector(chats map[int64]Chat, useCase MessageCollector,
	report func(domain.Message, error)) *TelegramCollector {
	return &TelegramCollector{chats: chats, useCase: useCase, report: report}
}

// Listen passes the client's updates to Handle until ctx ends.
func (c *TelegramCollector) Listen(ctx context.Context, client telegram.Client) error {
	return client.Listen(ctx, c.Handle)
}

// Handle maps one update; a service update or one of an unwatched chat is dropped.
func (c *TelegramCollector) Handle(ctx context.Context, update telegram.Update) {
	chat, ok := c.chats[update.ChatID]
	if !ok || update.Service {
		return
	}
	message := toMessage(chat, update)
	if err := c.useCase.Collect(ctx, message); err != nil && ctx.Err() == nil {
		c.report(message, err)
	}
}

// toMessage maps an update of a watched chat to a message.
func toMessage(chat Chat, update telegram.Update) domain.Message {
	return domain.Message{
		Key:       domain.MessageKey{ChatID: update.ChatID, MessageID: update.MessageID},
		Chat:      chat.Ref,
		ChatTitle: chat.Title,
		SentAt:    update.Date,
		Text:      update.Text,
		Link:      link(chat, update),
		Edited:    update.Edited,
	}
}

func link(chat Chat, update telegram.Update) string {
	if chat.Username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", chat.Username, update.MessageID)
	}
	return fmt.Sprintf("https://t.me/c/%d/%d", update.ChatID, update.MessageID)
}
