package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// heldLlm holds a judgment until it is released or its context ends.
type heldLlm struct {
	entered chan struct{}
	release chan struct{}
}

func newHeldLlm() *heldLlm {
	return &heldLlm{entered: make(chan struct{}, 1), release: make(chan struct{})}
}

func (l *heldLlm) Judge(ctx context.Context, _, _ string) (domain.Judgment, error) {
	l.entered <- struct{}{}
	select {
	case <-l.release:
		return domain.Judgment{Raw: "no"}, nil
	case <-ctx.Done():
		return domain.Judgment{}, ctx.Err()
	}
}

// downLlm never answers: every call fails as if LM Studio were not running.
type downLlm struct{}

func (downLlm) Judge(context.Context, string, string) (domain.Judgment, error) {
	return domain.Judgment{}, errors.New("connection refused")
}

func oneLiveMessage() *fakeTelegram {
	return &fakeTelegram{updates: []telegram.Update{{ChatID: 7001, MessageID: 4211, Date: time.Now(), Text: "a kids' bike"}}}
}

func TestServeSaysWhatItFinishesAndLetsTheMessageInWorkEnd(t *testing.T) {
	c, out, _ := newTestConsole(true)
	messages := &memoryMessages{}
	llm := newHeldLlm()
	matcher := newTestLive(t, c, messages)
	matcher.ports.Llm = llm
	matcher.settings.StopGrace = 5 * time.Second
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- matcher.serve(ctx, c, oneLiveMessage(), marketFound) }()
	<-llm.entered
	stop()
	require.Eventually(t, func() bool { return strings.Contains(c.written(out), "-> stopping") }, 5*time.Second, 5*time.Millisecond)
	close(llm.release)

	require.NoError(t, <-done)
	assert.Contains(t, out.String(), "-> stopping · finishing 1 message in work · up to 5s · Ctrl+C again to stop now\n")
	assert.Equal(t, 1, messages.processedCount(), "the message in work was finished after the stop")
}

func TestServeEndsTheMessageInWorkAfterTheGraceTime(t *testing.T) {
	c, _, _ := newTestConsole(true)
	messages := &memoryMessages{}
	llm := newHeldLlm()
	matcher := newTestLive(t, c, messages)
	matcher.ports.Llm = llm
	matcher.settings.StopGrace = 100 * time.Millisecond
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- matcher.serve(ctx, c, oneLiveMessage(), marketFound) }()
	<-llm.entered
	started := time.Now()
	stop()

	require.NoError(t, <-done)
	assert.Less(t, time.Since(started), 2*time.Second)
	assert.Zero(t, messages.processedCount(), "a message not finished in the grace time stays unprocessed")
}

func TestServeEndsARetryPauseAtTheStop(t *testing.T) {
	c, _, errOut := newTestConsole(true)
	messages := &memoryMessages{}
	matcher := newTestLive(t, c, messages)
	matcher.ports.Llm = downLlm{}
	matcher.settings.StopGrace = 30 * time.Second
	matcher.settings.RetryMaxPause = time.Minute
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- matcher.serve(ctx, c, oneLiveMessage(), marketFound) }()
	require.Eventually(t, func() bool { return strings.Contains(c.written(errOut), "LM Studio is not answering") },
		5*time.Second, 5*time.Millisecond)
	started := time.Now()
	stop()

	require.NoError(t, <-done)
	assert.Less(t, time.Since(started), time.Second, "the pause ended at the stop, not after the grace time")
	assert.Zero(t, messages.processedCount())
	assert.Contains(t, errOut.String(), "[warn] LM Studio is not answering · retry in 1s, 2s, 4s … up to 60s")
}

func TestServeReadsTheGapOnceTheConnectionIsBack(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	messages := &memoryMessages{}
	client := &fakeTelegram{}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- newTestLive(t, c, messages).serve(ctx, c, client, marketFound) }()
	require.Eventually(t, func() bool { return strings.Contains(c.written(out), "catch-up done") }, 5*time.Second, 5*time.Millisecond)
	client.connection(telegram.ConnectionLost)
	client.post(telegram.Update{ChatID: 7001, MessageID: 4400, Date: time.Now(), Text: "a kids' bike posted during the drop"})
	client.connection(telegram.ConnectionBack)
	require.Eventually(t, func() bool { return messages.processedCount() == 1 }, 5*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool { return strings.Contains(c.written(out), "gap read done") }, 5*time.Second, 5*time.Millisecond)
	stop()

	require.NoError(t, <-done)
	assert.Contains(t, c.written(errOut), "[warn] Telegram connection lost · reconnecting\n")
	assert.Contains(t, out.String(), "[ok] Telegram connection back · reading 1 chat from their cursors\n")
	assert.Contains(t, out.String(), "[ok] gap read done · 1 message to judge\n")
	assert.Len(t, client.sends(), 1, "the match of the gap reached the owner")
}

func TestServeSaysWhenTheQueueIsFull(t *testing.T) {
	c, _, errOut := newTestConsole(true)
	llm := newHeldLlm()
	matcher := newTestLive(t, c, &memoryMessages{})
	matcher.ports.Llm = llm
	matcher.workers = 1
	matcher.settings.Workers = 1
	client := &fakeTelegram{}
	for id := 1; id <= 15; id++ { // one in work, ten queued, the rest wait
		client.updates = append(client.updates, telegram.Update{ChatID: 7001, MessageID: id, Date: time.Now(), Text: "a kids' bike"})
	}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error)

	go func() { done <- matcher.serve(ctx, c, client, marketFound) }()
	require.Eventually(t, func() bool { return strings.Contains(c.written(errOut), "queue is full") }, 5*time.Second, 5*time.Millisecond)
	stop()

	require.NoError(t, <-done)
	assert.Equal(t, 1, strings.Count(errOut.String(), "[warn] queue is full (10) · reading new messages paused\n"))
}
