package yaml_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/settings/yaml"
)

// oneRequest is a valid request block; lines 4-7 of a config after header.
const oneRequest = "requests:\n  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market']\n"

func parseErrors(t *testing.T, text string) []yaml.FieldError {
	t.Helper()
	_, _, err := yaml.Parse([]byte(text))
	var configErrs *yaml.ConfigErrors
	require.True(t, errors.As(err, &configErrs), "got %v", err)
	return configErrs.Errors
}

type wantError struct {
	line   int
	field  string
	reason string
}

// problemCases are configs with one problem each and the error it gives.
var problemCases = []struct {
	name string
	text string
	want wantError
}{
	{"no requests", header, wantError{1, "requests", "at least one request"}},
	{"empty requests", header + "requests: []\n", wantError{4, "requests", "at least one request"}},
	{"no id", header + "requests:\n  - find: x\n    chats: ['@made_up_market']\n", wantError{5, "requests[0].id", "is required"}},
	{"no find", header + "requests:\n  - id: bike\n    chats: ['@made_up_market']\n", wantError{5, "requests[0].find", "is required"}},
	{"no chats", header + "requests:\n  - id: bike\n    find: x\n", wantError{5, "requests[0].chats", "at least one chat"}},
	{"empty chats", header + "requests:\n  - id: bike\n    find: x\n    chats: []\n", wantError{7, "requests[0].chats", "at least one chat"}},
	{"unknown root key", header + "colour: red\n" + oneRequest, wantError{4, "colour", "unknown key"}},
	{"unknown request key", header + oneRequest + "    model: x\n", wantError{8, "requests[0].model", "unknown key"}},
	{"unknown chat key", header + "requests:\n  - id: bike\n    find: x\n    chats:\n      - {ref: '@made_up_market', mute: true}\n",
		wantError{8, "requests[0].chats[0].mute", "unknown key"}},
	{"unknown depth key", header + "catch_up: {weeks: 1}\n" + oneRequest, wantError{4, "catch_up.weeks", "unknown key"}},
	{"wrong kind", header + "workers: many\n" + oneRequest, wantError{4, "workers", "whole number"}},
	{"zero depth", header + "catch_up: {days: 0}\n" + oneRequest, wantError{4, "catch_up.days", "at least 1"}},
	{"negative depth", header + "requests:\n  - id: bike\n    find: x\n    catch_up: {messages: -5}\n    chats: ['@made_up_market']\n",
		wantError{7, "requests[0].catch_up.messages", "at least 1"}},
	{"bad url", "owner: '@made_up_owner'\nlm_studio: {model: m, url: 'localhost:1234'}\n" + oneRequest, wantError{2, "lm_studio.url", "http or https"}},
	{"bad timeout", "owner: '@made_up_owner'\nlm_studio: {model: m, timeout: soon}\n" + oneRequest, wantError{2, "lm_studio.timeout", "duration"}},
	{"no model", "owner: '@made_up_owner'\nlm_studio: {url: 'http://localhost:1234/v1'}\n" + oneRequest, wantError{2, "lm_studio.model", "is required"}},
	{"no lm_studio", "owner: '@made_up_owner'\n" + oneRequest, wantError{1, "lm_studio.model", "is required"}},
	{"zero workers", header + "workers: 0\n" + oneRequest, wantError{4, "workers", "at least 1"}},
	{"no owner", "lm_studio: {model: m}\n" + oneRequest, wantError{1, "owner", "is required"}},
	{"bad owner", "owner: 'https://t.me/made_up_owner'\nlm_studio: {model: m}\n" + oneRequest, wantError{1, "owner", "not an owner"}},
	{"bad chat", header + "requests:\n  - id: bike\n    find: x\n    chats: [made_up_market]\n", wantError{7, "requests[0].chats[0]", "not a chat reference"}},
	{"chat without ref", header + "requests:\n  - id: bike\n    find: x\n    chats:\n      - catch_up: {days: 1}\n",
		wantError{8, "requests[0].chats[0].ref", "is required"}},
	{"api_id at root", header + "api_id: 1\n" + oneRequest, wantError{4, "api_id", "secrets go to .env"}},
	{"api_hash in lm_studio", "owner: '@made_up_owner'\nlm_studio: {model: m, api_hash: x}\n" + oneRequest,
		wantError{2, "lm_studio.api_hash", "secrets go to .env"}},
	{"session under an unknown key", header + "telegram:\n  session: x\n" + oneRequest, wantError{5, "telegram.session", "secrets go to .env"}},
}

func TestParseReportsEachProblemWithLineAndField(t *testing.T) {
	for _, tc := range problemCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := parseErrors(t, tc.text)

			assert.Contains(t, describe(errs), fmt.Sprintf("%d %s", tc.want.line, tc.want.field))
			for _, each := range errs {
				if each.Field == tc.want.field {
					assert.Contains(t, each.Reason, tc.want.reason)
				}
			}
		})
	}
}

func describe(errs []yaml.FieldError) string {
	var lines []string
	for _, each := range errs {
		lines = append(lines, fmt.Sprintf("%d %s: %s", each.Line, each.Field, each.Reason))
	}
	return strings.Join(lines, "\n")
}

func TestParseReportsASecretKeyOnlyOnce(t *testing.T) {
	errs := parseErrors(t, header+"api_hash: x\n"+oneRequest)

	assert.Equal(t, []yaml.FieldError{{Line: 4, Field: "api_hash", Reason: "secrets go to .env, not to tgscout.yaml"}}, errs)
}

func TestParseNamesTheFirstLineOfADuplicateID(t *testing.T) {
	errs := parseErrors(t, header+oneRequest+"  - id: bike\n    find: y\n    chats: ['@made_up_street']\n")

	assert.Equal(t, []yaml.FieldError{{Line: 8, Field: "requests[1].id", Reason: `"bike" is used twice (first at line 5)`}}, errs)
}

func TestParseListsEveryProblemAtOnceSortedByLine(t *testing.T) {
	errs := parseErrors(t, header+`requests:
  - id: bike
    find: x
    chats: []
  - id: bike
    find: y
    chats: ['@made_up_market']
api_hash: x
`)

	assert.Equal(t, []yaml.FieldError{
		{Line: 7, Field: "requests[0].chats", Reason: "a request needs at least one chat"},
		{Line: 8, Field: "requests[1].id", Reason: `"bike" is used twice (first at line 5)`},
		{Line: 11, Field: "api_hash", Reason: "secrets go to .env, not to tgscout.yaml"},
	}, errs)
}

func manyChats(count int) string {
	var chats []string
	for i := range count {
		chats = append(chats, fmt.Sprintf("'%d'", 1000+i))
	}
	return header + "requests:\n  - id: bike\n    find: x\n    chats: [" + strings.Join(chats, ", ") + "]\n"
}

func TestParseWarnsAboveTheDesignMaximumOfChats(t *testing.T) {
	_, warnings, err := yaml.Parse([]byte(manyChats(101)))

	require.NoError(t, err)
	assert.Equal(t, []string{"101 chats; the design maximum is 100"}, warnings)
}

func TestParseDoesNotWarnAtTheDesignMaximum(t *testing.T) {
	_, warnings, err := yaml.Parse([]byte(manyChats(100)))

	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestParseGivesTheLineOfBrokenYaml(t *testing.T) {
	_, _, err := yaml.Parse([]byte("requests:\n  - id: bike\n    find: a kids bike\n    chats: a: b\n"))

	var syntaxErr *yaml.ConfigSyntaxError
	require.True(t, errors.As(err, &syntaxErr), "got %v", err)
	assert.Equal(t, 4, syntaxErr.Line)
}
