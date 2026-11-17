package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// ErrLlmRejected marks an LLM error about the message itself (a client error,
// an answer that cannot be read): such a call is tried at most MaxRejectedTries
// times, while a call that gets no answer is tried without end.
var ErrLlmRejected = errors.New("the LLM rejected the call")

// MaxRejectedTries is how often a call the LLM rejects is tried before the
// decision is saved as «no».
const MaxRejectedTries = 10

// DeliveryReporter is told what the retries do, for the console.
type DeliveryReporter interface {
	// LlmDown is the first call with no answer while the LLM answered before.
	LlmDown(maxPause time.Duration)
	// LlmBack is the first answer after LlmDown.
	LlmBack()
	// GaveUp is a request saved as «no» after the LLM rejected every try.
	GaveUp(item domain.QueueItem, request domain.Request, tries int, err error)
	// SendFailed is a failed send that is tried again after the pause.
	SendFailed(match domain.Match, err error, pause time.Duration)
}

// Delivery is how MatchMessage retries: the pauses, how it waits and whom it tells.
// A zero Sleep waits on a timer; a nil Reporter tells nobody.
type Delivery struct {
	Backoff  domain.Backoff
	Sleep    func(ctx context.Context, pause time.Duration) error
	Reporter DeliveryReporter
}

func (d Delivery) sleep(ctx context.Context, pause time.Duration) error {
	if d.Sleep != nil {
		return d.Sleep(ctx, pause)
	}
	return SleepContext(ctx, pause)
}

func (d Delivery) reporter() DeliveryReporter {
	if d.Reporter == nil {
		return DeliveryFuncs{}
	}
	return d.Reporter
}

// SleepContext waits the pause, or less when the context ends first.
func SleepContext(ctx context.Context, pause time.Duration) error {
	timer := time.NewTimer(pause)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DeliveryFuncs is a DeliveryReporter made of functions; a nil one does nothing.
type DeliveryFuncs struct {
	OnLlmDown    func(maxPause time.Duration)
	OnLlmBack    func()
	OnGaveUp     func(item domain.QueueItem, request domain.Request, tries int, err error)
	OnSendFailed func(match domain.Match, err error, pause time.Duration)
}

// LlmDown calls OnLlmDown.
func (f DeliveryFuncs) LlmDown(maxPause time.Duration) {
	if f.OnLlmDown != nil {
		f.OnLlmDown(maxPause)
	}
}

// LlmBack calls OnLlmBack.
func (f DeliveryFuncs) LlmBack() {
	if f.OnLlmBack != nil {
		f.OnLlmBack()
	}
}

// GaveUp calls OnGaveUp.
func (f DeliveryFuncs) GaveUp(item domain.QueueItem, request domain.Request, tries int, err error) {
	if f.OnGaveUp != nil {
		f.OnGaveUp(item, request, tries, err)
	}
}

// SendFailed calls OnSendFailed.
func (f DeliveryFuncs) SendFailed(match domain.Match, err error, pause time.Duration) {
	if f.OnSendFailed != nil {
		f.OnSendFailed(match, err, pause)
	}
}

// llmHealth turns many failing workers into one "down" and one "back" report.
type llmHealth struct {
	mu   sync.Mutex
	down bool
}

// failed reports whether this failure is the one that says the LLM is down.
func (h *llmHealth) failed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	first := !h.down
	h.down = true
	return first
}

// answered reports whether this answer is the one that says the LLM is back.
func (h *llmHealth) answered() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	back := h.down
	h.down = false
	return back
}
