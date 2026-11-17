package tests

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cucumber/godog"

	notify "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/notify/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// ownerSlice wires the real MatchMessage and TelegramNotifierClient over a
// stub Telegram client and the memory repository.
type ownerSlice struct {
	repository *memoryRepository
	client     *sendingClient
	slept      []time.Duration
	warnings   []string
	matchErr   error
}

func (s *slice) ownerSteps(sc *godog.ScenarioContext) {
	o := &ownerSlice{repository: s.repository, client: &sendingClient{started: make(chan struct{}, 1)}}
	sc.Step(`^Telegram has not yet confirmed the send of a match$`, func() { o.client.block = true })
	sc.Step(`^Telegram answers a send with a wait of 5 seconds$`, func() {
		o.client.floods = []time.Duration{5 * time.Second}
	})
	sc.Step(`^the first two sends of a match fail$`, func() { o.client.failures = 2 })
	sc.Step(`^tgscout stops at that moment$`, o.stopDuringTheSend)
	sc.Step(`^tgscout sends a match$`, func() error { return o.match(context.Background()) })
	sc.Step(`^the message is not marked processed and the cursor has not moved past it$`, func() error {
		if !errors.Is(o.matchErr, context.Canceled) {
			return fmt.Errorf("the match ended with %v, want the stop", o.matchErr)
		}
		return o.repository.untouched(ownerKey)
	})
	sc.Step(`^the console shows two warning lines, the owner gets the match once and the message is marked processed after the send$`,
		o.sentAfterTwoFailures)
	sc.Step(`^it waits 5 seconds, sends again and marks the message processed after the send is confirmed$`, o.waitedAndSent)
}

var ownerKey = domain.MessageKey{ChatID: watchedChatID, MessageID: 4211}

func (o *ownerSlice) match(ctx context.Context) error {
	tracker := domain.NewMessageTracker()
	notifier := notify.NewTelegramNotifierClient(o.client, "@made_up_owner",
		notify.PrintFunc(func(domain.Match, string) error { return nil }), func(time.Duration) {},
		func(_ context.Context, d time.Duration) error { o.slept = append(o.slept, d); return nil })
	useCase := application.NewMatchMessage(o.repository, yesLlm{}, notifier, tracker, application.Delivery{
		Backoff: domain.Backoff{Max: time.Minute},
		Sleep:   func(context.Context, time.Duration) error { return nil },
		Reporter: application.DeliveryFuncs{OnSendFailed: func(match domain.Match, err error, pause time.Duration) {
			o.warnings = append(o.warnings, fmt.Sprintf("send failed: %s · %v · retry in %s", match.Request.ID, err, pause))
		}},
	})
	ref, err := domain.NewChatRef("@made_up_market")
	if err != nil {
		return err
	}
	tracker.Begin(ownerKey)
	_, o.matchErr = useCase.Match(ctx, domain.QueueItem{
		Message:  domain.Message{Key: ownerKey, Chat: ref, Text: "Selling a kids bike", Link: "https://t.me/made_up_market/4211"},
		Requests: []domain.Request{{ID: "bike", Find: "a kids bike", Chats: []domain.RequestChat{{Ref: ref}}}},
	})
	return nil
}

func (o *ownerSlice) stopDuringTheSend() error {
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- o.match(ctx) }()
	select {
	case <-o.client.started:
	case <-time.After(2 * time.Second):
		stop()
		return errors.New("the send never started")
	}
	stop()
	return <-done
}

func (o *ownerSlice) waitedAndSent() error {
	if o.matchErr != nil {
		return fmt.Errorf("the match failed: %w", o.matchErr)
	}
	if fmt.Sprint(o.slept) != "[5s]" {
		return fmt.Errorf("slept %v, want [5s]", o.slept)
	}
	if o.client.attempts() != 2 {
		return fmt.Errorf("%d sends tried, want 2", o.client.attempts())
	}
	if o.repository.processedCount() != 1 {
		return errors.New("the message is not marked processed")
	}
	return nil
}

func (o *ownerSlice) sentAfterTwoFailures() error {
	if o.matchErr != nil {
		return fmt.Errorf("the match failed: %w", o.matchErr)
	}
	want := "[send failed: bike · a made-up network error · retry in 1s send failed: bike · a made-up network error · retry in 2s]"
	if fmt.Sprint(o.warnings) != want {
		return fmt.Errorf("warnings %v, want %s", o.warnings, want)
	}
	if o.client.attempts() != 3 || o.client.delivered != 1 {
		return fmt.Errorf("%d sends tried and %d delivered, want 3 and 1", o.client.attempts(), o.client.delivered)
	}
	if o.repository.processedCount() != 1 {
		return errors.New("the message is not marked processed")
	}
	return nil
}

// sendingClient is the stub Telegram of the slice: a send can wait for the
// stop or answer FLOOD_WAIT first.
type sendingClient struct {
	telegram.Client
	mu        sync.Mutex
	block     bool
	floods    []time.Duration
	failures  int
	tries     int
	delivered int
	started   chan struct{}
}

func (c *sendingClient) SendToOwner(ctx context.Context, _, _ string) error {
	c.mu.Lock()
	c.tries++
	block := c.block
	var flood time.Duration
	if len(c.floods) > 0 {
		flood, c.floods = c.floods[0], c.floods[1:]
	}
	failed := c.tries <= c.failures
	if !failed && !block && flood == 0 {
		c.delivered++
	}
	c.mu.Unlock()
	if failed {
		return errors.New("a made-up network error")
	}
	if block {
		c.started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}
	if flood > 0 {
		return &telegram.FloodWait{Wait: flood}
	}
	return nil
}

func (c *sendingClient) attempts() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tries
}

type yesLlm struct{}

func (yesLlm) Judge(context.Context, string, string) (domain.Judgment, error) {
	return domain.Judgment{Matched: true, Raw: "yes"}, nil
}
