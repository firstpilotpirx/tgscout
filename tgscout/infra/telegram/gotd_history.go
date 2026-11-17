package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// ReadHistory pages messages.getHistory of a chat ResolveChat found.
func (c *gotdClient) ReadHistory(ctx context.Context, chatID int64, window HistoryWindow) (History, error) {
	c.mu.Lock()
	peer := c.peers[chatID]
	c.mu.Unlock()
	if peer == nil {
		return History{}, fmt.Errorf("chat %d was not found at start", chatID)
	}
	return pageHistory(ctx, func(ctx context.Context, offsetID int) ([]Update, error) {
		answer, err := c.client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer: peer, OffsetID: offsetID, Limit: historyPage, MinID: window.After})
		if err != nil {
			return nil, mapError(err)
		}
		modified, ok := answer.AsModified()
		if !ok {
			return nil, nil
		}
		var page []Update
		for _, message := range modified.GetMessages() {
			if update, ok := mapMessage(message, false); ok {
				page = append(page, update)
			}
		}
		return page, nil
	}, window)
}
