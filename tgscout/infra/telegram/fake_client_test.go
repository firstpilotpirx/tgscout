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
