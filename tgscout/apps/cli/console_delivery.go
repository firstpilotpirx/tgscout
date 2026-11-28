package main

import (
	"fmt"
	"time"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// The console lines of the retries, the reconnect and the stop (design D10).

func (c *console) warnLine(text string) {
	c.println(c.errOut, c.yellow.Sprint(c.marks.warn)+" "+text)
}

// llmDown says LM Studio stopped answering and how the retries wait.
func (c *console) llmDown(maxPause time.Duration) {
	c.warnLine(fmt.Sprintf("LM Studio is not answering · retry in 1s, 2s, 4s … up to %s · messages wait in the queue",
		seconds(maxPause)))
}

// llmBack says LM Studio answers again and how many messages wait.
func (c *console) llmBack(waiting int) {
	c.done(fmt.Sprintf("LM Studio is back · %s waiting", countThousands(waiting, "message")))
}

// queueFull says the queue is full and reading new messages waits.
func (c *console) queueFull(size int) {
	c.warnLine(fmt.Sprintf("queue is full (%s) · reading new messages paused", thousands(size)))
}

// gaveUp shows a request saved as «no» after LM Studio rejected every try.
// It never shows the message text.
func (c *console) gaveUp(item domain.QueueItem, request domain.Request, tries int, err error) {
	c.warnLine(fmt.Sprintf("judge gave up after %d tries: %s on %s · saved as no · %v",
		tries, request.ID, where(item.Message), err))
}

// sendFailed shows a send that is tried again after the pause.
func (c *console) sendFailed(match domain.Match, err error, pause time.Duration) {
	c.warnLine(fmt.Sprintf("send failed: %s on %s · %v · retry in %s · %s",
		match.Request.ID, where(match.Message), err, seconds(pause), unprocessed))
}

func (c *console) connectionLost() { c.warnLine("Telegram connection lost · reconnecting") }

func (c *console) connectionBack(chats int) {
	c.done(fmt.Sprintf("Telegram connection back · reading %s from their cursors", count(chats, "chat")))
}

// gapRead says how many messages of the gap went to the judge, and which chats it could not read.
func (c *console) gapRead(handed int, failed []collect.ChatReport) {
	c.done(fmt.Sprintf("gap read done · %s to judge", countThousands(handed, "message")))
	for _, report := range failed {
		name := report.Chat.Title
		if name == "" {
			name = report.Chat.Ref.String()
		}
		c.failBlock(fmt.Sprintf("gap read: %s · cannot read the history: %v", name, report.Err), "next start reads it again")
	}
}

// gapChatRefreshed says the gap read looked up again a stored chat Telegram no longer knew.
func (c *console) gapChatRefreshed(chat collect.Chat) { c.refreshedLine("gap read", chat) }

// stopping says the stop began, how many messages are still judged and how long it waits for them.
func (c *console) stopping(busy int, grace time.Duration) {
	c.step(fmt.Sprintf("stopping · finishing %s in work · up to %s · Ctrl+C again to stop now",
		countThousands(busy, "message"), seconds(grace)))
}

// stopped closes the run and says how many messages stay in the queue.
func (c *console) stopped(queued int) {
	if queued == 0 {
		c.done("stopped")
		return
	}
	c.done(fmt.Sprintf("stopped · %s in the queue left unprocessed, catch-up reads them next start",
		countThousands(queued, "message")))
}

// seconds writes a pause in whole seconds: 60s, not 1m0s.
func seconds(d time.Duration) string {
	if d < time.Second {
		return d.String()
	}
	return fmt.Sprintf("%ds", int(d.Round(time.Second)/time.Second))
}
