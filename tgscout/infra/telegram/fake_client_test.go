//go:build telegramfake

package telegram

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func newTestFake(t *testing.T, scenario string) (*fakeClient, string) {
	t.Helper()
	sent := filepath.Join(t.TempDir(), "sent.jsonl")
	parsed, err := parseFakeScenario([]byte(strings.Replace(scenario, "{", `{"sent_file": "`+sent+`", `, 1)))
	require.NoError(t, err)
	return &fakeClient{scenario: parsed}, sent
}

func TestFakeSendToOwnerWritesEachSendToTheSentFile(t *testing.T) {
	fake, sent := newTestFake(t, `{"contacts": ["@made_up_owner"]}`)

	require.NoError(t, fake.SendToOwner(context.Background(), "@made_up_owner", "Match: one\nline two"))
	require.NoError(t, fake.SendToOwner(context.Background(), "@made_up_owner", "Match: two"))

	data, err := os.ReadFile(sent)
	require.NoError(t, err)
	assert.Equal(t, `{"to":"@made_up_owner","text":"Match: one\nline two"}`+"\n"+
		`{"to":"@made_up_owner","text":"Match: two"}`+"\n", string(data))
}

func TestFakeSendToOwnerFailsWhenTheScenarioSaysSo(t *testing.T) {
	fake, sent := newTestFake(t, `{"send_fails": true}`)

	err := fake.SendToOwner(context.Background(), "@made_up_owner", "Match: one")

	require.Error(t, err)
	assert.NoFileExists(t, sent)
}

func TestFakeSendToOwnerAsksToWaitOnce(t *testing.T) {
	fake, _ := newTestFake(t, `{"send_flood_wait_seconds": 2}`)

	var flood *FloodWait
	require.ErrorAs(t, fake.SendToOwner(context.Background(), "@made_up_owner", "Match: one"), &flood)
	assert.Equal(t, 2*time.Second, flood.Wait)
	require.NoError(t, fake.SendToOwner(context.Background(), "@made_up_owner", "Match: one"))
}

func TestFakeFindOwnerTellsTheOwnerState(t *testing.T) {
	open, _ := newTestFake(t, `{"contacts": ["@made_up_owner"]}`)
	closed, _ := newTestFake(t, `{"contacts": ["@made_up_owner"], "owner_chat_closed": true}`)

	ready, err := open.FindOwner(context.Background(), "@made_up_owner")
	require.NoError(t, err)
	missing, err := open.FindOwner(context.Background(), "@someone_else")
	require.NoError(t, err)
	noChat, err := closed.FindOwner(context.Background(), "@made_up_owner")
	require.NoError(t, err)

	assert.Equal(t, []OwnerState{OwnerReady, OwnerMissing, OwnerNoChat}, []OwnerState{ready, missing, noChat})
}

func TestFakeChatInfoGivesTheScenarioTitle(t *testing.T) {
	scenario, err := parseFakeScenario([]byte(`{"chats": {"*": "joined"}, "titles": {"@made_up_market": "Made-up market"}}`))
	require.NoError(t, err)

	assert.Equal(t, "Made-up market", scenario.info("@made_up_market").Title)
	assert.Empty(t, scenario.info("@made_up_street").Title)
}

func TestFakeListenDropsAPrivateMessage(t *testing.T) {
	fake, _ := newTestFake(t, `{"chats": {"*": "joined"}, "updates": [
		{"chat": "@made_up_stranger", "id": 1, "text": "hello", "private": true},
		{"chat": "@made_up_market", "id": 2, "text": "Selling a bike"}]}`)
	ctx, stop := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer stop()
	var got []int

	_ = fake.Listen(ctx, func(_ context.Context, update Update) { got = append(got, update.MessageID) })

	assert.Equal(t, []int{2}, got)
}

func TestFakeReadHistoryGivesTheChatsMessagesInTheWindow(t *testing.T) {
	fake, _ := newTestFake(t, `{"chats": {"*": "joined"}, "history": [
		{"chat": "@made_up_market", "id": 1, "text": "old", "age_minutes": 5000},
		{"chat": "@made_up_market", "id": 2, "text": "Selling a bike", "age_minutes": 60},
		{"chat": "@made_up_street", "id": 3, "text": "other chat", "age_minutes": 30},
		{"chat": "@made_up_market", "id": 4, "text": "Selling a lamp", "age_minutes": 10}]}`)
	id := fake.scenario.info("@made_up_market").ID

	history, err := fake.ReadHistory(context.Background(), id,
		HistoryWindow{Limit: 500, Since: time.Now().Add(-72 * time.Hour)})

	require.NoError(t, err)
	require.Len(t, history.Messages, 2)
	assert.Equal(t, 2, history.Messages[0].MessageID)
	assert.Equal(t, "Selling a lamp", history.Messages[1].Text)
	assert.Equal(t, id, history.Messages[1].ChatID)
	assert.True(t, history.Gap)
}

func TestFakeReadHistoryAsksToWaitOnce(t *testing.T) {
	fake, _ := newTestFake(t, `{"history_flood_wait_seconds": 1}`)

	var flood *FloodWait
	_, err := fake.ReadHistory(context.Background(), 7, HistoryWindow{Limit: 5})
	require.ErrorAs(t, err, &flood)
	assert.Equal(t, time.Second, flood.Wait)
	_, err = fake.ReadHistory(context.Background(), 7, HistoryWindow{Limit: 5})
	require.NoError(t, err)
}

func TestFakeReadHistoryPausesFirst(t *testing.T) {
	fake, _ := newTestFake(t, `{"history_delay_ms": 100}`)
	began := time.Now()

	_, err := fake.ReadHistory(context.Background(), 7, HistoryWindow{Limit: 5})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(began), 100*time.Millisecond)
}
