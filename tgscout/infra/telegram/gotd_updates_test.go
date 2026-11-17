package telegram

import (
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/assert"
)

func TestMapsANewChannelMessage(t *testing.T) {
	update, ok := mapMessage(&tg.Message{ID: 4211, PeerID: &tg.PeerChannel{ChannelID: 1234567890},
		Date: 1767362520, Message: "Selling a 16-inch kids' bike"}, false)

	assert.True(t, ok)
	assert.Equal(t, Update{ChatID: 1234567890, MessageID: 4211, Date: time.Unix(1767362520, 0),
		Text: "Selling a 16-inch kids' bike"}, update)
}

func TestMapsAnEditedGroupMessage(t *testing.T) {
	update, ok := mapMessage(&tg.Message{ID: 98, PeerID: &tg.PeerChat{ChatID: 555}, Message: "Desk lamp"}, true)

	assert.True(t, ok)
	assert.Equal(t, int64(555), update.ChatID)
	assert.True(t, update.Edited)
}

func TestMapsAServiceMessage(t *testing.T) {
	update, ok := mapMessage(&tg.MessageService{ID: 7, PeerID: &tg.PeerChannel{ChannelID: 1}}, false)

	assert.True(t, ok)
	assert.True(t, update.Service)
	assert.Empty(t, update.Text)
}

func TestMapsACaptionAsTheText(t *testing.T) {
	update, ok := mapMessage(&tg.Message{ID: 8, PeerID: &tg.PeerChannel{ChannelID: 1},
		Media: &tg.MessageMediaPhoto{}, Message: "A desk lamp, see the photo"}, false)

	assert.True(t, ok)
	assert.Equal(t, "A desk lamp, see the photo", update.Text)
}

func TestSkipsAPrivateMessageOfAUser(t *testing.T) {
	_, ok := mapMessage(&tg.Message{ID: 9, PeerID: &tg.PeerUser{UserID: 1}, Message: "hi"}, false)

	assert.False(t, ok)
}

func TestBareChatIDDropsTheMinusPrefixes(t *testing.T) {
	assert.Equal(t, int64(1234567890), bareChatID("-1001234567890"))
	assert.Equal(t, int64(555), bareChatID("-555"))
	assert.Equal(t, int64(555), bareChatID("555"))
}

func TestChannelInfoGivesTheIDAndTheUsername(t *testing.T) {
	info := chatInfo(&tg.ContactsResolvedPeer{Chats: []tg.ChatClass{&tg.Channel{ID: 42, Username: "made_up_market",
		Title: "Made-up market"}}})

	assert.Equal(t, ChatInfo{State: ChatJoined, ID: 42, Username: "made_up_market", Title: "Made-up market"}, info)
}

func TestAPrivateChatIsOpenOnlyWithATopMessage(t *testing.T) {
	assert.False(t, hasTopMessage(nil))
	assert.False(t, hasTopMessage([]tg.DialogClass{&tg.Dialog{TopMessage: 0}}))
	assert.True(t, hasTopMessage([]tg.DialogClass{&tg.Dialog{TopMessage: 17}}))
}

func TestFirstUserPeerKeepsTheAccessHash(t *testing.T) {
	users := []tg.UserClass{&tg.User{ID: 5, AccessHash: 50}, &tg.User{ID: 6, AccessHash: 60}}

	assert.Equal(t, &tg.InputPeerUser{UserID: 5, AccessHash: 50}, firstUserPeer(users, 0))
	assert.Equal(t, &tg.InputPeerUser{UserID: 6, AccessHash: 60}, firstUserPeer(users, 6))
	assert.Nil(t, firstUserPeer(users, 7))
}
