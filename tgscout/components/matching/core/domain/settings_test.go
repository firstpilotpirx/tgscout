package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func chat(t *testing.T, text string) domain.RequestChat {
	t.Helper()
	ref, err := domain.NewChatRef(text)
	if err != nil {
		t.Fatal(err)
	}
	return domain.RequestChat{Ref: ref, CatchUp: domain.CatchUp{Messages: 500, Days: 3}}
}

func TestDistinctChatsCountsEachChatOnce(t *testing.T) {
	settings := domain.Settings{Requests: []domain.Request{
		{ID: "bike", Chats: []domain.RequestChat{chat(t, "@made_up_market"), chat(t, "-1001")}},
		{ID: "lamp", Chats: []domain.RequestChat{chat(t, "https://t.me/Made_Up_Market"), chat(t, "@made_up_street")}},
	}}

	assert.Equal(t, 3, settings.DistinctChats())
}

func TestDistinctChatsOfNoRequestsIsZero(t *testing.T) {
	assert.Equal(t, 0, domain.Settings{}.DistinctChats())
}

func TestRequestsOfGivesEveryRequestOfTheChatInConfigOrder(t *testing.T) {
	settings := domain.Settings{Requests: []domain.Request{
		{ID: "bike", Chats: []domain.RequestChat{chat(t, "@made_up_market")}},
		{ID: "lamp", Chats: []domain.RequestChat{chat(t, "@made_up_street")}},
		{ID: "desk", Chats: []domain.RequestChat{chat(t, "https://t.me/Made_Up_Market")}},
	}}
	ref, _ := domain.NewChatRef("@made_up_market")

	requests := settings.RequestsOf(ref)

	assert.Equal(t, []string{"bike", "desk"}, []string{requests[0].ID, requests[1].ID})
	assert.Len(t, requests, 2)
}

func TestRequestsOfAnUnwatchedChatIsEmpty(t *testing.T) {
	ref, _ := domain.NewChatRef("@made_up_other")
	assert.Empty(t, domain.Settings{}.RequestsOf(ref))
}
