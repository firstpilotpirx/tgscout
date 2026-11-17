package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// lmCall is one chat completion tgscout asked the fake LM Studio.
type lmCall struct {
	Model    string
	System   string
	Request  string
	Message  string
	Original string // the user message as sent
}

// fakeLmStudio is a made-up OpenAI-compatible server: it answers per
// request text and records every call (design § Test levels).
type fakeLmStudio struct {
	server  *httptest.Server
	mu      sync.Mutex
	calls   []lmCall
	answers map[string]string // request text → answer; "" is the answer to any other
	down    bool              // every call gets 503
	delay   time.Duration     // each answer comes this late, or never when the caller leaves first
}

func newFakeLmStudio() *fakeLmStudio {
	lm := &fakeLmStudio{answers: map[string]string{"": "no"}}
	lm.server = httptest.NewServer(http.HandlerFunc(lm.serve))
	return lm
}

func (lm *fakeLmStudio) url() string { return lm.server.URL + "/v1" }

func (lm *fakeLmStudio) serve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if r.URL.Path != "/v1/chat/completions" || json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Messages) != 2 {
		http.Error(w, "made-up LM Studio: bad request", http.StatusBadRequest)
		return
	}
	user := body.Messages[1].Content
	call := lmCall{Model: body.Model, System: body.Messages[0].Content, Original: user}
	request, message, _ := strings.Cut(strings.TrimPrefix(user, "Request: "), "\nMessage: ")
	call.Request, call.Message = request, message
	lm.mu.Lock()
	lm.calls = append(lm.calls, call)
	answer, ok := lm.answers[request]
	if !ok {
		answer = lm.answers[""]
	}
	down, delay := lm.down, lm.delay
	lm.mu.Unlock()
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return // tgscout gave the call up
		}
	}
	if down {
		http.Error(w, "made-up LM Studio is down", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{ // a lost answer fails the scenario on its own
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": answer}}},
	})
}

// setDown makes LM Studio go away or come back while tgscout runs.
func (lm *fakeLmStudio) setDown(down bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.down = down
}

// setDelay makes every later answer come this late.
func (lm *fakeLmStudio) setDelay(delay time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.delay = delay
}

// callsAbout gives the calls about one request text, in order.
func (lm *fakeLmStudio) callsAbout(request string) []lmCall {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	var calls []lmCall
	for _, each := range lm.calls {
		if each.Request == request {
			calls = append(calls, each)
		}
	}
	return calls
}

func (lm *fakeLmStudio) allCalls() []lmCall {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return append([]lmCall(nil), lm.calls...)
}
