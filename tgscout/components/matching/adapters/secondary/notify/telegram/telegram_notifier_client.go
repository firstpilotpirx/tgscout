// Package telegram tells the owner about a match in a private Telegram
// message from tgscout's own account, then on the console.
package telegram

import (
	"context"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// MatchPrinter shows a sent match and its text on the console; the app's
// console implements it, since only the app may use the color library.
type MatchPrinter interface {
	PrintMatch(match domain.Match, text string) error
}

// PrintFunc is a MatchPrinter made of one function, so the app passes its
// console's method and the adapter never holds an app type.
type PrintFunc func(match domain.Match, text string) error

// PrintMatch calls the function.
func (f PrintFunc) PrintMatch(match domain.Match, text string) error { return f(match, text) }

// TelegramNotifierClient sends each match to the owner over the shared
// MTProto client, waiting out every FLOOD_WAIT.
type TelegramNotifierClient struct {
	client  telegram.Client
	owner   string
	printer MatchPrinter
	waiting func(time.Duration)
	sleep   func(context.Context, time.Duration) error
}

// NewTelegramNotifierClient builds the notifier; owner is the config text,
// waiting reports a FLOOD_WAIT and sleep waits it.
func NewTelegramNotifierClient(client telegram.Client, owner string, printer MatchPrinter,
	waiting func(time.Duration), sleep func(context.Context, time.Duration) error) *TelegramNotifierClient {
	return &TelegramNotifierClient{client: client, owner: owner, printer: printer, waiting: waiting, sleep: sleep}
}

// Notify returns once Telegram confirmed the send; only then the match goes
// to the console. A console write that fails after the send is dropped: the
// owner has the match, and sending it again would repeat it.
func (n *TelegramNotifierClient) Notify(ctx context.Context, match domain.Match) error {
	text := domain.NotificationText(match)
	_, err := telegram.Retry(ctx, n.waiting, n.sleep, func() (struct{}, error) {
		return struct{}{}, n.client.SendToOwner(ctx, n.owner, text)
	})
	if err != nil {
		return err
	}
	_ = n.printer.PrintMatch(match, text) // the send is confirmed; a lost console line is not a lost match
	return nil
}
