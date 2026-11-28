package yaml_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/settings/yaml"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

const header = "owner: '@made_up_owner'\nlm_studio:\n  model: example-model\n"

func parse(t *testing.T, text string) domain.Settings {
	t.Helper()
	settings, _, err := yaml.Parse([]byte(text))
	require.NoError(t, err)
	return settings
}

func depth(settings domain.Settings, request, chat int) domain.CatchUp {
	return settings.Requests[request].Chats[chat].CatchUp
}

func TestParseRootDepthAppliesWhenNothingOverridesIt(t *testing.T) {
	settings := parse(t, header+`catch_up: {messages: 500, days: 3}
requests:
  - id: bike
    find: a kids bike
    chats: ['@made_up_market']
`)

	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 3}, depth(settings, 0, 0))
}

func TestParseRequestOverridesTheRoot(t *testing.T) {
	settings := parse(t, header+`catch_up: {messages: 500, days: 3}
requests:
  - id: bike
    find: a kids bike
    catch_up: {days: 7}
    chats: ['@made_up_market', '-1001']
`)

	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 7}, depth(settings, 0, 0))
	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 7}, depth(settings, 0, 1))
}

func TestParseChatOverridesItsRequest(t *testing.T) {
	settings := parse(t, header+`requests:
  - id: bike
    find: a kids bike
    catch_up: {days: 7}
    chats:
      - ref: '@made_up_market'
        catch_up: {messages: 2000}
      - '@made_up_street'
`)

	assert.Equal(t, domain.CatchUp{Messages: 2000, Days: 7}, depth(settings, 0, 0))
	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 7}, depth(settings, 0, 1))
}

func TestParseSameChatInTwoRequestsKeepsTwoDepths(t *testing.T) {
	settings := parse(t, header+`requests:
  - id: bike
    find: a kids bike
    catch_up: {days: 7}
    chats: ['@made_up_market']
  - id: lamp
    find: a desk lamp
    chats:
      - ref: '@made_up_market'
        catch_up: {messages: 50, days: 1}
`)

	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 7}, depth(settings, 0, 0))
	assert.Equal(t, domain.CatchUp{Messages: 50, Days: 1}, depth(settings, 1, 0))
}

func TestParseBuiltInDepthWhenTheRootSetsNone(t *testing.T) {
	settings := parse(t, header+`requests:
  - id: bike
    find: a kids bike
    chats: ['@made_up_market', '@made_up_street']
`)

	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 3}, depth(settings, 0, 0))
	assert.Equal(t, domain.CatchUp{Messages: 500, Days: 3}, depth(settings, 0, 1))
}

func TestParseDefaultsFillWhatTheFileLeavesOut(t *testing.T) {
	settings := parse(t, header+`requests:
  - id: bike
    find: a kids bike
    chats: ['@made_up_market']
`)

	assert.Equal(t, "@made_up_owner", settings.Owner.String())
	assert.Equal(t, 2, settings.Workers)
	assert.Equal(t, 1000, settings.QueueSize)
	assert.Equal(t, domain.LmStudio{URL: "http://localhost:1234/v1", Model: "example-model", Timeout: 60 * time.Second},
		settings.LmStudio)
	assert.Equal(t, 60*time.Second, settings.RetryMaxPause)
	assert.Equal(t, 10*time.Second, settings.StopGrace)
	assert.Equal(t, 5*time.Minute, settings.StatusInterval)
	assert.Equal(t, time.Second, settings.HistoryPause)
	assert.Equal(t, "bike", settings.Requests[0].ID)
	assert.Equal(t, "a kids bike", settings.Requests[0].Find)
}

func TestParseTakesEverySetValue(t *testing.T) {
	settings := parse(t, `owner: 1234567
workers: 4
queue_size: 50
retry_max_pause: 30s
stop_grace: 2s
status_interval: 10s
history_pause: 3s
lm_studio: {url: 'http://127.0.0.1:9000/v1', model: other-model, timeout: 90s}
requests:
  - id: bike
    find: a kids bike
    chats: ['@made_up_market']
`)

	assert.Equal(t, "1234567", settings.Owner.String())
	assert.Equal(t, 4, settings.Workers)
	assert.Equal(t, 50, settings.QueueSize)
	assert.Equal(t, 30*time.Second, settings.RetryMaxPause)
	assert.Equal(t, 2*time.Second, settings.StopGrace)
	assert.Equal(t, 10*time.Second, settings.StatusInterval)
	assert.Equal(t, 3*time.Second, settings.HistoryPause)
	assert.Equal(t, domain.LmStudio{URL: "http://127.0.0.1:9000/v1", Model: "other-model", Timeout: 90 * time.Second},
		settings.LmStudio)
}

func TestParseAcceptsEveryFormOfChatReference(t *testing.T) {
	settings := parse(t, header+`requests:
  - id: bike
    find: a kids bike
    chats: ['@made_up_market', 'https://t.me/made_up_street', -1001234567890]
`)

	var refs []string
	for _, chat := range settings.Requests[0].Chats {
		refs = append(refs, chat.Ref.String())
	}
	assert.Equal(t, []string{"@made_up_market", "https://t.me/made_up_street", "-1001234567890"}, refs)
}
