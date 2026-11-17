// Package channel runs the matcher workers over the channel queue.
package channel

import (
	"context"
	"sync"

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

// RunWorkers runs the workers until ctx ends and returns once all stopped.
func RunWorkers(ctx context.Context, workers int, queue Receiver, matcher Matcher, report Reporter) {
	var wait sync.WaitGroup
	for range max(workers, 1) {
		wait.Add(1)
		go func() {
			defer wait.Done()
			work(ctx, queue, matcher, report)
		}()
	}
	wait.Wait()
}

func work(ctx context.Context, queue Receiver, matcher Matcher, report Reporter) {
	for {
		item, err := queue.Receive(ctx)
		if err != nil {
			return
		}
		outcome, err := matcher.Match(ctx, item)
		if ctx.Err() != nil {
			return
		}
		for _, unclear := range outcome.Unclear {
			report.Unclear(item, unclear)
		}
		if err != nil {
			report.Failed(item, err)
		}
	}
}
