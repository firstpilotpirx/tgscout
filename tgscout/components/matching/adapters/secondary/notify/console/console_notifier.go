// Package console tells the owner about a match on the console.
package console

import (
	"context"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// MatchPrinter draws a match line; the app's console implements it, since
// only the app may use the color library.
type MatchPrinter interface {
	PrintMatch(match domain.Match) error
}

// ConsoleNotifier hands each match to the printer.
type ConsoleNotifier struct{ printer MatchPrinter }

// NewConsoleNotifier builds the notifier over a printer.
func NewConsoleNotifier(printer MatchPrinter) *ConsoleNotifier {
	return &ConsoleNotifier{printer: printer}
}

// Notify prints the match.
func (n *ConsoleNotifier) Notify(_ context.Context, match domain.Match) error {
	return n.printer.PrintMatch(match)
}

// PrintFunc is a MatchPrinter made of one function, so the app passes its
// console's method and the adapter never holds an app type.
type PrintFunc func(match domain.Match) error

// PrintMatch calls the function.
func (f PrintFunc) PrintMatch(match domain.Match) error { return f(match) }
