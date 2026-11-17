//go:build integration

package openai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/llm/openai"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

type call struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	Messages    []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// fakeServer answers with the given texts in turn and records every call.
func fakeServer(t *testing.T, status int, answers ...string) (*httptest.Server, *[]call) {
	var mu sync.Mutex
	var calls []call
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		var body call
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		mu.Lock()
		calls = append(calls, body)
		answer := answers[min(len(calls), len(answers))-1]
		mu.Unlock()
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": answer}}},
		})
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func client(server *httptest.Server, timeout time.Duration) *openai.OpenAiLlmClient {
	return openai.NewOpenAiLlmClient(domain.LmStudio{URL: server.URL + "/v1", Model: "example-model", Timeout: timeout})
}

func judge(t *testing.T, answers ...string) (domain.Judgment, []call) {
	server, calls := fakeServer(t, http.StatusOK, answers...)
	judgment, err := client(server, time.Second).Judge(context.Background(), "a desk lamp", "Desk lamp, 10 EUR")
	require.NoError(t, err)
	return judgment, *calls
}

func TestJudgeSendsTheModelTheRulesAndTheTextsApart(t *testing.T) {
	_, calls := judge(t, "yes")

	require.Len(t, calls, 1)
	assert.Equal(t, "example-model", calls[0].Model)
	assert.Zero(t, calls[0].Temperature)
	require.Len(t, calls[0].Messages, 2)
	assert.Equal(t, "system", calls[0].Messages[0].Role)
	assert.NotContains(t, calls[0].Messages[0].Content, "desk lamp")
	assert.Equal(t, "user", calls[0].Messages[1].Role)
	assert.Contains(t, calls[0].Messages[1].Content, "Request: a desk lamp")
	assert.Contains(t, calls[0].Messages[1].Content, "Message: Desk lamp, 10 EUR")
}

func TestJudgeReadsYesAndNo(t *testing.T) {
	yes, _ := judge(t, "yes")
	no, _ := judge(t, "No")
	assert.Equal(t, domain.Judgment{Matched: true, Raw: "yes"}, yes)
	assert.Equal(t, domain.Judgment{Matched: false, Raw: "No"}, no)
}

func TestJudgeReadsAYesWithPunctuation(t *testing.T) {
	judgment, _ := judge(t, " Yes. ")
	assert.True(t, judgment.Matched)
}

func TestJudgeRemovesAThinkBlock(t *testing.T) {
	judgment, _ := judge(t, "<think>The lamp has a clamp.</think>\nyes")
	assert.True(t, judgment.Matched)
}

func TestJudgeAsksOnceMoreAfterAnUnclearAnswer(t *testing.T) {
	judgment, calls := judge(t, "It might be", "yes")

	require.Len(t, calls, 2)
	assert.NotEqual(t, calls[0].Messages[0].Content, calls[1].Messages[0].Content, "the second ask is stricter")
	assert.True(t, judgment.Matched)
	assert.False(t, judgment.Unclear)
}

func TestJudgeGivesUpAfterTwoUnclearAnswers(t *testing.T) {
	judgment, calls := judge(t, "It might be", "Hard to say")

	assert.Len(t, calls, 2)
	assert.Equal(t, domain.Judgment{Unclear: true, Raw: "Hard to say"}, judgment)
}

func TestJudgeReportsAnErrorStatus(t *testing.T) {
	server, _ := fakeServer(t, http.StatusServiceUnavailable, "")

	_, err := client(server, time.Second).Judge(context.Background(), "a desk lamp", "Desk lamp")

	require.Error(t, err)
	assert.Equal(t, server.URL+"/v1 answered 503 Service Unavailable", err.Error())
}

func TestJudgeReportsATimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	t.Cleanup(server.Close)

	_, err := client(server, 50*time.Millisecond).Judge(context.Background(), "a desk lamp", "Desk lamp")

	require.Error(t, err)
	assert.Equal(t, server.URL+"/v1 did not answer within 50ms", err.Error())
}

func TestJudgeReportsAnAddressThatDoesNotAnswer(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	_, err := client(server, time.Second).Judge(context.Background(), "a desk lamp", "Desk lamp")

	require.Error(t, err)
	assert.Equal(t, "cannot reach "+server.URL+"/v1", err.Error())
	assert.Error(t, errors.Unwrap(err), "the cause stays for errors.Is")
}
