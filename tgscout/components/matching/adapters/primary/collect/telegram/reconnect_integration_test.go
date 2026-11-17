//go:build integration

package telegram_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// gatedClient holds each history read until the test lets it go.
type gatedClient struct {
	telegram.Client
	mu    sync.Mutex
	reads int
	gate  chan struct{}
	began chan struct{}
}

func (c *gatedClient) ReadHistory(ctx context.Context, chatID int64, _ telegram.HistoryWindow) (telegram.History, error) {
	c.mu.Lock()
	c.reads++
	c.mu.Unlock()
	c.began <- struct{}{}
	select {
	case <-c.gate:
	case <-ctx.Done():
		return telegram.History{}, ctx.Err()
	}
	return telegram.History{Messages: []telegram.Update{{ChatID: chatID, MessageID: 1, Text: "posted during the drop"}}}, nil
}

func (c *gatedClient) readCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

type reconnectReports struct {
	mu    sync.Mutex
	lines []string
	done  chan int
}

func (r *reconnectReports) funcs() collect.ReconnectReports {
	return collect.ReconnectReports{
		OnLost: func() { r.add("lost") },
		OnBack: func(chats int) { r.add("back", chats) },
		OnGapDone: func(handed int, failed []collect.ChatReport) {
			r.add("gap read", handed, len(failed))
			r.done <- handed
		},
	}
}

func (r *reconnectReports) add(line string, numbers ...int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range numbers {
		line += " " + string(rune('0'+n))
	}
	r.lines = append(r.lines, line)
}

func (r *reconnectReports) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

func runReconnector(t *testing.T, useCase *recordingCatchUp, client telegram.Client, reports *reconnectReports) (*collect.Reconnector, context.CancelFunc) {
	t.Helper()
	collector := collect.NewHistoryCollector(useCase, collect.HistoryReports{}, noSleep)
	reconnector := collect.NewReconnector(useCase, collector, client, historyChats(t)[:1], reports.funcs())
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		reconnector.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		stop()
		<-done
	})
	return reconnector, stop
}

func TestALostConnectionHoldsTheCursorsAndIsReportedOnce(t *testing.T) {
	useCase := &recordingCatchUp{}
	reports := &reconnectReports{done: make(chan int, 1)}
	reconnector, _ := runReconnector(t, useCase, &historyClient{}, reports)

	reconnector.Handle(telegram.ConnectionLost)

	assert.Eventually(t, func() bool { return len(reports.all()) == 1 }, time.Second, 5*time.Millisecond)
	assert.Equal(t, []string{"lost"}, reports.all())
	useCase.mu.Lock()
	defer useCase.mu.Unlock()
	assert.Equal(t, []int64{42}, useCase.paused)
}

func TestABackRunsOneGapRead(t *testing.T) {
	useCase := &recordingCatchUp{}
	reports := &reconnectReports{done: make(chan int, 1)}
	client := &historyClient{history: map[int64]telegram.History{
		42: {Messages: []telegram.Update{{ChatID: 42, MessageID: 11, Text: "posted during the drop"}}},
	}}
	reconnector, _ := runReconnector(t, useCase, client, reports)

	reconnector.Handle(telegram.ConnectionLost)
	reconnector.Handle(telegram.ConnectionBack)

	select {
	case handed := <-reports.done:
		assert.Equal(t, 1, handed)
	case <-time.After(time.Second):
		t.Fatal("no gap read after the connection was back")
	}
	assert.Equal(t, []string{"lost", "back 1", "gap read 1 0"}, reports.all())
}

func TestABackDuringAGapReadRunsOneMoreRead(t *testing.T) {
	useCase := &recordingCatchUp{}
	reports := &reconnectReports{done: make(chan int, 2)}
	client := &gatedClient{gate: make(chan struct{}), began: make(chan struct{}, 4)}
	reconnector, _ := runReconnector(t, useCase, client, reports)

	reconnector.Handle(telegram.ConnectionBack)
	<-client.began
	reconnector.Handle(telegram.ConnectionLost)
	reconnector.Handle(telegram.ConnectionBack)
	reconnector.Handle(telegram.ConnectionLost)
	reconnector.Handle(telegram.ConnectionBack)
	client.gate <- struct{}{}
	<-reports.done
	<-client.began
	client.gate <- struct{}{}
	<-reports.done

	require.Never(t, func() bool { return client.readCount() > 2 }, 100*time.Millisecond, 10*time.Millisecond)
	assert.Equal(t, 2, client.readCount(), "the Backs during a read make one more read")
}
