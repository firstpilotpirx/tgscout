//go:build telegramfake

package telegram

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFakeScenarioReadsEveryField(t *testing.T) {
	scenario, err := parseFakeScenario([]byte(`{"session_valid": true, "account": "Made Up Scout",
		"code": "24680", "password": "made-up-pass",
		"chats": {"@made_up_market": "joined", "@made_up_river": "not_joined"},
		"contacts": ["@made_up_owner"], "flood_wait_seconds": 1}`))

	require.NoError(t, err)
	assert.True(t, scenario.SessionValid)
	assert.Equal(t, "Made Up Scout", scenario.Account)
	assert.Equal(t, "24680", scenario.Code)
	assert.Equal(t, "made-up-pass", scenario.Password)
	assert.Equal(t, ChatJoined, scenario.state("@made_up_market"))
	assert.Equal(t, ChatNotJoined, scenario.state("@made_up_river"))
	assert.Equal(t, ChatMissing, scenario.state("@no_such_chat"))
	assert.Equal(t, time.Second, scenario.floodWait())
	assert.Equal(t, []string{"@made_up_owner"}, scenario.Contacts)
}

func TestParseFakeScenarioRefusesBrokenJSON(t *testing.T) {
	_, err := parseFakeScenario([]byte(`{`))

	assert.Error(t, err)
}

func TestFakeScenarioStarStateCoversEveryOtherChat(t *testing.T) {
	scenario, err := parseFakeScenario([]byte(`{"chats": {"*": "joined", "@made_up_river": "not_joined"}}`))

	require.NoError(t, err)
	assert.Equal(t, ChatJoined, scenario.state("-1001234567890"))
	assert.Equal(t, ChatNotJoined, scenario.state("@made_up_river"))
}

func TestParseFakeScenarioReadsTheUpdates(t *testing.T) {
	scenario, err := parseFakeScenario([]byte(`{"chats": {"@made_up_market": "joined"},
		"updates": [{"chat": "@made_up_market", "id": 4211, "text": "Selling a bike", "after_ms": 20},
		{"chat": "@made_up_market", "id": 4211, "text": "Selling a red bike", "edited": true},
		{"chat": "@made_up_market", "id": 4212, "service": true}]}`))

	require.NoError(t, err)
	require.Len(t, scenario.Updates, 3)
	assert.Equal(t, fakeUpdate{Chat: "@made_up_market", ID: 4211, Text: "Selling a bike", AfterMs: 20}, scenario.Updates[0])
	assert.True(t, scenario.Updates[1].Edited)
	assert.True(t, scenario.Updates[2].Service)
}

func TestFakeChatInfoGivesEachChatItsOwnMadeUpID(t *testing.T) {
	scenario, err := parseFakeScenario([]byte(`{"chats": {"*": "joined"}}`))
	require.NoError(t, err)

	market, numeric := scenario.info("@made_up_market"), scenario.info("-1001234567890")

	assert.Equal(t, "made_up_market", market.Username)
	assert.Equal(t, int64(1234567890), numeric.ID)
	assert.Empty(t, numeric.Username)
	assert.NotEqual(t, market.ID, scenario.info("@made_up_street").ID)
	assert.Equal(t, market.ID, scenario.info("@made_up_market").ID)
}
