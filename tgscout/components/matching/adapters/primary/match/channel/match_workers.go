// Package channel runs the matcher workers over the channel queue.
package channel

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Receiver gives the next queued item.
type Receiver interface {
	Receive(ctx context.Context) (domain.QueueItem, error)
}

// Matcher is the match use case.
type Matcher interface {
	Match(ctx context.Context, item domain.QueueItem) (application.Outcome, error)
}

// Reporter prints what a judgment leaves besides matches.
type Reporter interface {
	Unclear(item domain.QueueItem, unclear application.Unclear)
	Failed(item domain.QueueItem, err error)
}

// Busy counts the workers judging a message right now.
type Busy struct{ n atomic.Int64 }

// Count is the number of messages in work.
func (b *Busy) Count() int { return int(b.n.Load()) }

func (b *Busy) add(delta int64) {
	if b != nil {
		b.n.Add(delta)
	}
}

// Pool is what the workers run on: how many, the queue, the matcher, the
// reports and, when set, the busy count.
type Pool struct {
	Workers int
	Queue   Receiver
	Matcher Matcher
	Report  Reporter
	Busy    *Busy
}

// RunWorkers runs the workers and returns once all ended. A worker takes no
// message after stop ends; a message in work goes on until it is judged or
// work ends.
func RunWorkers(stop, work context.Context, pool Pool) {
	var wait sync.WaitGroup
	for range max(pool.Workers, 1) {
		wait.Add(1)
		go func() {
			defer wait.Done()
			pool.run(stop, work)
		}()
	}
	wait.Wait()
}

func (p Pool) run(stop, work context.Context) {
	for {
		item, err := p.Queue.Receive(stop)
		if err != nil {
			return
		}
		p.Busy.add(1)
		outcome, err := p.Matcher.Match(work, item)
		p.Busy.add(-1)
		for _, unclear := range outcome.Unclear {
			p.Report.Unclear(item, unclear)
		}
		if err != nil && stop.Err() == nil {
			p.Report.Failed(item, err)
		}
	}
}

// ReportFuncs is a Reporter made of two functions, so the app passes its
// console's methods and the adapter never holds an app type.
type ReportFuncs struct {
	OnUnclear func(item domain.QueueItem, unclear application.Unclear)
	OnFailed  func(item domain.QueueItem, err error)
}

// Unclear calls OnUnclear.
func (r ReportFuncs) Unclear(item domain.QueueItem, unclear application.Unclear) {
	r.OnUnclear(item, unclear)
}

// Failed calls OnFailed.
func (r ReportFuncs) Failed(item domain.QueueItem, err error) { r.OnFailed(item, err) }
