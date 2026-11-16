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
