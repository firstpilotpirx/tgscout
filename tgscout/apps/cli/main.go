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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, plainOutput(color.NoColor, os.Getenv("FORCE_COLOR")))
	stop()
	os.Exit(code)
}
