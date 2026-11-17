package main

import (
	"context"
	"os"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// exitSecondSignal is the exit code of a second Ctrl+C while tgscout stops.
const exitSecondSignal = 130

// stopOnSignals cancels the run at the first signal and exits at once at the
// second (design D7).
func stopOnSignals(signals <-chan os.Signal, cancel context.CancelFunc, exit func(int)) {
	<-signals
	cancel()
	<-signals
	exit(exitSecondSignal)
}

// runDetached runs the Telegram client on a context the stop does not end, so
// a send in work can still reach Telegram during the grace time; the work gets
// a context that ends at the stop, and the client ends when the work returns.
func runDetached(stop context.Context, client telegram.Client, work func(context.Context) error) error {
	return client.Run(context.WithoutCancel(stop), func(clientCtx context.Context) error {
		workCtx, cancel := context.WithCancel(clientCtx)
		defer cancel()
		defer context.AfterFunc(stop, cancel)()
		return work(workCtx)
	})
}
