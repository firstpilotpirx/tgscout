//go:build integration

package console_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/notify/console"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

type bufferPrinter struct{ out bytes.Buffer }

func (p *bufferPrinter) PrintMatch(match domain.Match) error {
	_, err := fmt.Fprintf(&p.out, "match %s %s\n", match.Request.ID, match.Message.Link)
	return err
}

func TestNotifierHandsEachMatchToThePrinter(t *testing.T) {
	printer := &bufferPrinter{}
	notifier := console.NewConsoleNotifier(printer)
	match := domain.Match{
		Request: domain.Request{ID: "kids-bike"},
		Message: domain.Message{Link: "https://t.me/made_up_market/4211"},
	}

	require.NoError(t, notifier.Notify(context.Background(), match))

	assert.Equal(t, "match kids-bike https://t.me/made_up_market/4211\n", printer.out.String())
}
