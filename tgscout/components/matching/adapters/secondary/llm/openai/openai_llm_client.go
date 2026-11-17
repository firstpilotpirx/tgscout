// Package openai asks the LLM through an OpenAI-compatible API such as LM Studio's.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

const (
	rules = "You decide whether a Telegram message matches a person's request. " +
		"The user message holds the request and the message. Answer only yes or no."
	stricterRules = rules + " Reply with exactly one word: yes or no."
	maxTokens     = 256
)

var (
	thinkBlock  = regexp.MustCompile(`(?s)<think>.*?</think>`)
	punctuation = regexp.MustCompile(`[^a-z]`)
)

// OpenAiLlmClient judges a message with one chat completion per question.
type OpenAiLlmClient struct {
	base  string
	url   string
	model string
	http  *http.Client
}

// NewOpenAiLlmClient builds the client for the configured LM Studio.
func NewOpenAiLlmClient(settings domain.LmStudio) *OpenAiLlmClient {
	base := strings.TrimRight(settings.URL, "/")
	return &OpenAiLlmClient{
		base:  base,
		url:   base + "/chat/completions",
		model: settings.Model,
		http:  &http.Client{Timeout: settings.Timeout},
	}
}

// Judge asks whether the text matches the request; an unclear answer is
// asked once more with stricter rules, then returned as unclear.
func (c *OpenAiLlmClient) Judge(ctx context.Context, find, text string) (domain.Judgment, error) {
	var judgment domain.Judgment
	for _, system := range []string{rules, stricterRules} {
		raw, err := c.ask(ctx, system, "Request: "+find+"\nMessage: "+text)
		if err != nil {
			return domain.Judgment{}, err
		}
		judgment = read(raw)
		if !judgment.Unclear {
			return judgment, nil
		}
	}
	return judgment, nil
}

func read(raw string) domain.Judgment {
	answer := punctuation.ReplaceAllString(strings.ToLower(thinkBlock.ReplaceAllString(raw, "")), "")
	switch answer {
	case "yes":
		return domain.Judgment{Matched: true, Raw: raw}
	case "no":
		return domain.Judgment{Raw: raw}
	default:
		return domain.Judgment{Unclear: true, Raw: strings.TrimSpace(raw)}
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Messages    []message `json:"messages"`
}

type completionResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

func (c *OpenAiLlmClient) ask(ctx context.Context, system, user string) (string, error) {
	body, err := json.Marshal(completionRequest{Model: c.model, MaxTokens: maxTokens,
		Messages: []message{{Role: "system", Content: system}, {Role: "user", Content: user}}})
	if err != nil {
		return "", fmt.Errorf("build the LLM request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build the LLM request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return "", reachError(c.base, c.http.Timeout, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("%s answered %s", c.base, response.Status)
	}
	var completion completionResponse
	if err := json.NewDecoder(response.Body).Decode(&completion); err != nil {
		return "", fmt.Errorf("read the LLM answer: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("the LLM answer holds no choice")
	}
	return completion.Choices[0].Message.Content, nil
}
