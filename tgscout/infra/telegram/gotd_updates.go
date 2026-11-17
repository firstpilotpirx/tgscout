package telegram

import (
	"context"
	"sync"
	"time"

	"github.com/gotd/td/tg"
)

// updateSink passes the dispatcher's messages to the function Listen set.
type updateSink struct {
	mu sync.Mutex
	f  func(context.Context, Update)
}

func (s *updateSink) set(f func(context.Context, Update)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.f = f
}

func (s *updateSink) deliver(ctx context.Context, message tg.MessageClass, edited bool) error {
	s.mu.Lock()
	f := s.f
	s.mu.Unlock()
	if update, ok := mapMessage(message, edited); ok && f != nil {
		f(ctx, update)
	}
	return nil
}

func (s *updateSink) register(dispatcher tg.UpdateDispatcher) {
	dispatcher.OnNewMessage(func(ctx context.Context, _ tg.Entities, u *tg.UpdateNewMessage) error {
		return s.deliver(ctx, u.Message, false)
	})
	dispatcher.OnNewChannelMessage(func(ctx context.Context, _ tg.Entities, u *tg.UpdateNewChannelMessage) error {
		return s.deliver(ctx, u.Message, false)
	})
	dispatcher.OnEditMessage(func(ctx context.Context, _ tg.Entities, u *tg.UpdateEditMessage) error {
		return s.deliver(ctx, u.Message, true)
	})
	dispatcher.OnEditChannelMessage(func(ctx context.Context, _ tg.Entities, u *tg.UpdateEditChannelMessage) error {
		return s.deliver(ctx, u.Message, true)
	})
}

// mapMessage turns a message of a group or a channel into an Update; a
// message of a private chat with a user gives false.
func mapMessage(message tg.MessageClass, edited bool) (Update, bool) {
	switch message := message.(type) {
	case *tg.Message:
		chatID, ok := chatOf(message.PeerID)
		return Update{ChatID: chatID, MessageID: message.ID, Date: time.Unix(int64(message.Date), 0),
			Text: message.Message, Edited: edited}, ok
	case *tg.MessageService:
		chatID, ok := chatOf(message.PeerID)
		return Update{ChatID: chatID, MessageID: message.ID, Date: time.Unix(int64(message.Date), 0),
			Service: true, Edited: edited}, ok
	}
	return Update{}, false
}

func chatOf(peer tg.PeerClass) (int64, bool) {
	switch peer := peer.(type) {
	case *tg.PeerChannel:
		return peer.ChannelID, true
	case *tg.PeerChat:
		return peer.ChatID, true
	}
	return 0, false
}
