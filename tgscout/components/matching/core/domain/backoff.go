package domain

import "time"

// firstPause is the pause before the second try.
const firstPause = time.Second

// Backoff is the rule of the pauses between tries: 1 s, then twice the pause
// before, never more than Max.
type Backoff struct{ Max time.Duration }

// Pause gives the pause after the given failed try, counted from 1.
func (b Backoff) Pause(try int) time.Duration {
	pause := firstPause
	for n := 1; n < try && pause < b.Max; n++ {
		pause *= 2
	}
	return min(pause, max(b.Max, firstPause))
}
