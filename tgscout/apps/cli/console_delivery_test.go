package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func TestTheLlmDownLineNamesThePausesAndTheCap(t *testing.T) {
	c, out, errOut := newTestConsole(true)

	c.llmDown(time.Minute)

	assert.Equal(t, "[warn] LM Studio is not answering · retry in 1s, 2s, 4s … up to 60s · messages wait in the queue\n", errOut.String())
	assert.Empty(t, out.String())
}

func TestTheLlmDownLineWithSymbols(t *testing.T) {
	c, _, errOut := newTestConsole(false)

	c.llmDown(90 * time.Second)

	assert.Contains(t, errOut.String(), "⚠")
	assert.Contains(t, errOut.String(), " LM Studio is not answering · retry in 1s, 2s, 4s … up to 90s · messages wait in the queue\n")
}

func TestTheLlmBackLineCountsTheWaitingMessagesInThousands(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.llmBack(1000)
	c.llmBack(1)

	assert.Equal(t, "[ok] LM Studio is back · 1,000 messages waiting\n[ok] LM Studio is back · 1 message waiting\n", out.String())
}

func TestTheQueueFullLine(t *testing.T) {
	c, _, errOut := newTestConsole(true)

	c.queueFull(1000)

	assert.Equal(t, "[warn] queue is full (1,000) · reading new messages paused\n", errOut.String())
}

func TestTheGaveUpLineNamesTheRequestAndTheMessage(t *testing.T) {
	c, _, errOut := newTestConsole(true)
	item := domain.QueueItem{Message: madeUpMessage(t, "@made_up_market", 4610, "a text never shown")}

	c.gaveUp(item, domain.Request{ID: "bike-16"}, 10, errors.New("http://127.0.0.1:1234/v1 answered 400 Bad Request"))

	assert.Equal(t, "[warn] judge gave up after 10 tries: bike-16 on @made_up_market/4610 · saved as no · "+
		"http://127.0.0.1:1234/v1 answered 400 Bad Request\n", errOut.String())
	assert.NotContains(t, errOut.String(), "a text never shown")
}

func TestTheSendFailedLineNamesTheNextPause(t *testing.T) {
	c, _, errOut := newTestConsole(true)
	match := domain.Match{Message: madeUpMessage(t, "@made_up_market", 4529, "a text never shown"), Request: domain.Request{ID: "bike-16"}}

	c.sendFailed(match, errors.New("cannot reach Telegram"), 2*time.Second)

	assert.Equal(t, "[warn] send failed: bike-16 on @made_up_market/4529 · cannot reach Telegram · retry in 2s · "+
		"the message stays unprocessed\n", errOut.String())
}

func TestTheConnectionLines(t *testing.T) {
	c, out, errOut := newTestConsole(true)

	c.connectionLost()
	c.connectionBack(3)
	c.gapRead(42, nil)

	assert.Equal(t, "[warn] Telegram connection lost · reconnecting\n", errOut.String())
	assert.Equal(t, "[ok] Telegram connection back · reading 3 chats from their cursors\n[ok] gap read done · 42 messages to judge\n",
		out.String())
}

func TestAGapReadThatFailedForAChatSaysSo(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	ref, _ := domain.NewChatRef("@made_up_market")

	c.gapRead(1, []collect.ChatReport{{Chat: collect.Chat{Ref: ref}, Err: errors.New("cannot reach Telegram")}})

	assert.Equal(t, "[ok] gap read done · 1 message to judge\n", out.String())
	assert.Equal(t, "[fail] gap read: @made_up_market · cannot read the history: cannot reach Telegram\n"+
		"  next start reads it again\n", errOut.String())
}

func TestTheStoppingAndStoppedLines(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.stopping(2, 10*time.Second)
	c.stopped(1018)
	c.stopped(0)

	assert.Equal(t, "-> stopping · finishing 2 messages in work · up to 10s · Ctrl+C again to stop now\n"+
		"[ok] stopped · 1,018 messages in the queue left unprocessed, catch-up reads them next start\n"+
		"[ok] stopped\n", out.String())
}
