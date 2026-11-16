package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Exit codes of tgscout.
const (
	exitOK    = 0
	exitError = 1 // a config or database error
	exitUsage = 2 // an unknown flag or a stray argument
)

const defaultConfig = "./tgscout.yaml"

// usageError is a wrong flag or argument: it prints the usage and exits with exitUsage.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }

// options are the flags of the root command.
type options struct {
	config string
	db     string
}

// run parses args, runs tgscout until ctx ends and returns the exit code.
func run(ctx context.Context, args []string, out, errOut io.Writer, plain bool) int {
	con := newConsole(out, errOut, plain)
	code := exitOK
	cmd := newRootCommand(func(opts options) { code = start(ctx, opts, con) })
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	if err := cmd.Execute(); err != nil {
		var usage usageError
		if !errors.As(err, &usage) {
			usage = usageError{err}
		}
		con.fail(usage.Error(), "")
		_, _ = fmt.Fprint(errOut, "\n"+cmd.UsageString()) // nothing to do when stderr is gone
		return exitUsage
	}
	return code
}

func newRootCommand(startFn func(options)) *cobra.Command {
	var opts options
	cmd := &cobra.Command{
		Use:           "tgscout",
		Long:          "tgscout watches Telegram chats and writes to you when a message matches.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageError{fmt.Errorf("unknown argument %q", args[0])}
			}
			return nil
		},
		Run: func(_ *cobra.Command, _ []string) { startFn(opts) },
	}
	cmd.Flags().StringVar(&opts.config, "config", defaultConfig, "config file")
	cmd.Flags().StringVar(&opts.db, "db", "", "SQLite file (default: tgscout.db next to the config)")
	cmd.Flags().BoolP("help", "h", false, "show this help")
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	return cmd
}
