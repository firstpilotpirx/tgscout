//go:build integration

package telegram_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notify "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/notify/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// stubClient answers each send from a list and records what was sent; the
// other calls of the port stay unused.
type stubClient struct {
	telegram.Client
	answers []error
	sent    []string
	log     *[]string
}

func (c *stubClient) SendToOwner(_ context.Context, owner, text string) error {
	*c.log = append(*c.log, "send")
	var answer error
	if len(c.answers) > 0 {
		answer, c.answers = c.answers[0], c.answers[1:]
	}
	if answer == nil {
		c.sent = append(c.sent, owner+"\n"+text)
	}
	return answer
}

type recordingPrinter struct {
	log   *[]string
	texts []string
	err   error
}

func (p *recordingPrinter) PrintMatch(_ domain.Match, text string) error {
	*p.log = append(*p.log, "print")
	p.texts = append(p.texts, text)
	return p.err
}

type fixture struct {
	client   *stubClient
	printer  *recordingPrinter
	reported []time.Duration
	slept    []time.Duration
	notifier *notify.TelegramNotifierClient
	log      []string
}

func newFixture(answers ...error) *fixture {
	f := &fixture{}
	f.client = &stubClient{answers: answers, log: &f.log}
	f.printer = &recordingPrinter{log: &f.log}
	f.notifier = notify.NewTelegramNotifierClient(f.client, "@made_up_owner", f.printer,
		func(d time.Duration) { f.reported = append(f.reported, d) },
		func(_ context.Context, d time.Duration) error { f.slept = append(f.slept, d); return nil })
	return f
}

func bikeMatch(t *testing.T) domain.Match {
	t.Helper()
	ref, err := domain.NewChatRef("@made_up_market")
	require.NoError(t, err)
	return domain.Match{
		Request: domain.Request{ID: "kids-bike", Find: "a kids' bike"},
		Message: domain.Message{Chat: ref, ChatTitle: "Made-up market", Text: "Selling a bike, 80 €",
			Link: "https://t.me/made_up_market/4211"},
	}
}

func TestNotifySendsTheTextToTheOwnerThenPrintsIt(t *testing.T) {
	f := newFixture()
	match := bikeMatch(t)

	require.NoError(t, f.notifier.Notify(context.Background(), match))

	text := domain.NotificationText(match)
	assert.Equal(t, []string{"@made_up_owner\n" + text}, f.client.sent)
	assert.Equal(t, []string{text}, f.printer.texts)
	assert.Equal(t, []string{"send", "print"}, f.log)
}

func TestNotifyWaitsAFloodWaitAndSendsAgain(t *testing.T) {
	f := newFixture(&telegram.FloodWait{Wait: 5 * time.Second})

	require.NoError(t, f.notifier.Notify(context.Background(), bikeMatch(t)))

	assert.Equal(t, []time.Duration{5 * time.Second}, f.reported)
	assert.Equal(t, []time.Duration{5 * time.Second}, f.slept)
	assert.Equal(t, []string{"send", "send", "print"}, f.log)
	assert.Len(t, f.client.sent, 1)
}

func TestNotifyReturnsAFailedSendAndPrintsNothing(t *testing.T) {
	refused := errors.New("cannot reach Telegram")
	f := newFixture(refused)

	err := f.notifier.Notify(context.Background(), bikeMatch(t))

	require.ErrorIs(t, err, refused)
	assert.Empty(t, f.printer.texts)
}

func TestNotifyDropsAConsoleErrorAfterAConfirmedSend(t *testing.T) {
	f := newFixture()
	f.printer.err = errors.New("console closed")

	err := f.notifier.Notify(context.Background(), bikeMatch(t))

	require.NoError(t, err)
	assert.Len(t, f.client.sent, 1)
	assert.True(t, strings.HasPrefix(f.printer.texts[0], "Match: kids-bike"))
}
