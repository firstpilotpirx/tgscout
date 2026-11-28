package telegram

import (
	"context"
	"sync"
	"time"
)

// Pacer keeps at least Pause between the starts of two history calls, across
// every reader of one client (design D6). The pause runs between call
// starts, so a slow call uses up part of it; a Pause of 0 never waits.
type Pacer struct {
	Pause time.Duration
	now   func() time.Time
	sleep func(context.Context, time.Duration) error
	mu    sync.Mutex
	last  time.Time // the start of the last call; zero before the first
}

// NewPacer makes a pacer on the real clock.
func NewPacer(pause time.Duration) *Pacer {
	return &Pacer{Pause: pause, now: time.Now, sleep: SleepContext}
}

// Wait returns once a call may start: Pause after the start of the one
// before, or at once for the first call or when the pause is off. It holds
// the pacer while it sleeps, so readers take their turns.
func (p *Pacer) Wait(ctx context.Context) error {
	if p == nil || p.Pause <= 0 {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.last.IsZero() {
		if wait := p.last.Add(p.Pause).Sub(p.now()); wait > 0 {
			if err := p.sleep(ctx, wait); err != nil {
				return err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p.last = p.now()
	return nil
}
