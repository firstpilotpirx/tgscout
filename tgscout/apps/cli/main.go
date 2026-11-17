// Command tgscout watches Telegram chats and writes to the owner when a
// message matches a standing request.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go stopOnSignals(signals, cancel, os.Exit)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, plainOutput(color.NoColor, os.Getenv("FORCE_COLOR")))
	cancel()
	os.Exit(code)
}
