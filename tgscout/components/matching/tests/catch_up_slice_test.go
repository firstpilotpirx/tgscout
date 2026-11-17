package tests

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	match "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/match/channel"
	queue "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

var catchUpNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

var defaultDepth = domain.CatchUp{Messages: 500, Days: 3}

// catchUpSlice runs the history collector through the real use cases over a
// made-up chat history and the memory repository.
type catchUpSlice struct {
	repository *memoryRepository
	llm        *slowLlm
	ref        domain.ChatRef
	requests   []domain.Request
	history    []telegram.Update
	reports    []collect.ChatReport
	summary    *collect.Summary
}

func newCatchUpSlice() *catchUpSlice {
	ref, _ := domain.NewChatRef("@made_up_market")
	return &catchUpSlice{repository: newMemoryRepository(), llm: &slowLlm{}, ref: ref,
		requests: []domain.Request{{ID: "bike", Find: "a kids bike", Chats: []domain.RequestChat{{Ref: ref, CatchUp: defaultDepth}}}}}
}

func (c *catchUpSlice) steps(sc *godog.ScenarioContext) {
	sc.Step(`^the owner starts tgscout$`, c.run)
	sc.Step(`^catch-up reads (?:a|that) chat(?: of that request)? for the first time$`, c.run)
	sc.Step(`^catch-up reaches it$`, c.run)
	c.depthSteps(sc)
	c.depthPerRequestSteps(sc)
	c.processedSteps(sc)
}

// depthSteps: where catch-up starts and stops, and the gap.
func (c *catchUpSlice) depthSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a watched chat with no cursor and (\d+) messages posted in the last day$`, func(count int) {
		c.post(count, 0, time.Minute)
	})
	sc.Step(`^tgscout judges only the newest (\d+) of them$`, func(count int) error {
		return c.judgedFrom(len(c.history)-count+1, len(c.history))
	})
	sc.Step(`^a chat whose cursor is at a message of 12 days ago and newer messages over all those days$`, func() {
		c.repository.cursor[watchedChatID] = 1
		c.repository.cursorAt[watchedChatID] = catchUpNow.Add(-12 * 24 * time.Hour)
		c.post(12*24, 1, time.Hour) // ids 2..289, one per hour, the newest an hour ago
	})
	sc.Step(`^tgscout judges only the messages of the last 3 days$`, func() error {
		since := catchUpNow.Add(-3 * 24 * time.Hour)
		first := 0
		for _, each := range c.history {
			if first == 0 && !each.Date.Before(since) {
				first = each.MessageID
			}
		}
		return c.judgedFrom(first, c.history[len(c.history)-1].MessageID)
	})
	sc.Step(`^the console line of that chat says it has a gap$`, func() error {
		if len(c.reports) != 1 || !c.reports[0].Gap {
			return fmt.Errorf("the chat reports were %+v, want one with a gap", c.reports)
		}
		return nil
	})
}

func (c *catchUpSlice) depthPerRequestSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a request with a catch-up depth of 7 days$`, func() {
		c.requests[0].Chats[0].CatchUp = domain.CatchUp{Messages: 500, Days: 7}
		c.postAt(1, 8*24*time.Hour, "eight days old")
		c.postAt(2, 6*24*time.Hour, "six days old")
	})
	sc.Step(`^it reads back 7 days for that request$`, func() error { return c.askedTexts("six days old") })
	sc.Step(`^a chat of a request with a catch-up depth of 2,000 messages$`, func() {
		c.requests[0].Chats[0].CatchUp = domain.CatchUp{Messages: 2000, Days: 3}
		c.post(2500, 0, 30*time.Second)
	})
	sc.Step(`^it reads up to 2,000 messages$`, func() error {
		return c.judgedFrom(len(c.history)-2000+1, len(c.history))
	})
	sc.Step(`^a chat bound to a request with 7 days and a request with 3 days, and a message of 5 days ago$`, func() {
		c.requests = []domain.Request{
			{ID: "week", Find: "a kids bike", Chats: []domain.RequestChat{{Ref: c.ref, CatchUp: domain.CatchUp{Messages: 500, Days: 7}}}},
			{ID: "three", Find: "a desk lamp", Chats: []domain.RequestChat{{Ref: c.ref, CatchUp: defaultDepth}}},
		}
		c.postAt(1, 5*24*time.Hour, "Selling a kids bike")
	})
	sc.Step(`^the message is judged against the 7-day request only$`, func() error {
		if finds := c.llm.finds; strings.Join(finds, ",") != "a kids bike" {
			return fmt.Errorf("the message was judged against %q, want only %q", finds, "a kids bike")
		}
		return nil
	})
}

func (c *catchUpSlice) processedSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a message after the cursor that is already marked processed$`, func() {
		c.repository.cursor[watchedChatID] = 1
		c.repository.cursorAt[watchedChatID] = catchUpNow.Add(-time.Hour)
		c.repository.processed[domain.MessageKey{ChatID: watchedChatID, MessageID: 2}] = true
		c.repository.sent[domain.MessageKey{ChatID: watchedChatID, MessageID: 2}] = 1
		c.postAt(2, 30*time.Minute, "already judged")
		c.postAt(3, 20*time.Minute, "a new one")
	})
	sc.Step(`^it is skipped and no second match is sent$`, func() error {
		if err := c.askedTexts("a new one"); err != nil {
			return err
		}
		if sent := c.repository.sent[domain.MessageKey{ChatID: watchedChatID, MessageID: 2}]; sent != 1 {
			return fmt.Errorf("the processed message was sent %d times", sent)
		}
		if c.summary == nil || c.summary.Counts.Skipped != 1 {
			return fmt.Errorf("the summary was %+v, want one skipped", c.summary)
		}
		return nil
	})
}

// post adds count messages after the message id from, one each step back
// from now, the newest one step ago.
func (c *catchUpSlice) post(count, from int, step time.Duration) {
	for each := 1; each <= count; each++ {
		age := time.Duration(count-each+1) * step
		c.postAt(from+each, age, fmt.Sprintf("message %d", from+each))
	}
}

func (c *catchUpSlice) postAt(id int, age time.Duration, text string) {
	c.history = append(c.history, telegram.Update{ChatID: watchedChatID, MessageID: id, Date: catchUpNow.Add(-age), Text: text})
}

// run catches the chat up with two workers judging, and returns once the
// summary is given.
func (c *catchUpSlice) run() error {
	settings := staticSettings{domain.Settings{Requests: c.requests}}
	tracker := domain.NewMessageTracker()
	messages := queue.NewChannelMessageQueue(100)
	collectUseCase := application.NewCollectMessage(settings, c.repository, messages, tracker)
	catchUp := application.NewCatchUp(settings, c.repository, collectUseCase, tracker, func() time.Time { return catchUpNow })
	matchUseCase := application.NewMatchMessage(c.repository, c.llm, nopNotifier{}, tracker, application.Delivery{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		match.RunWorkers(ctx, 2, messages, matchUseCase, nopReporter{})
		close(stopped)
	}()
	collector := collect.NewHistoryCollector(catchUp, collect.HistoryReports{
		OnChat: func(report collect.ChatReport) { c.reports = append(c.reports, report) },
		OnDone: func(summary collect.Summary) { c.summary = &summary },
	}, func(context.Context, time.Duration) error { return nil })
	chats := []collect.Chat{{ID: watchedChatID, Ref: c.ref, Username: "made_up_market"}}
	err := collector.CatchUp(ctx, historyStub{history: c.history}, chats)
	cancel()
	<-stopped
	return err
}

// judgedFrom tells whether exactly the messages first..last were judged.
func (c *catchUpSlice) judgedFrom(first, last int) error {
	var want []string
	for _, each := range c.history {
		if each.MessageID >= first && each.MessageID <= last {
			want = append(want, each.Text)
		}
	}
	got := c.llm.texts()
	if len(got) != len(want) || !sameTexts(got, want) {
		return fmt.Errorf("%d messages were judged, want %d (ids %d..%d)", len(got), len(want), first, last)
	}
	return nil
}

func (c *catchUpSlice) askedTexts(texts ...string) error {
	if got := c.llm.texts(); !sameTexts(got, texts) {
		return fmt.Errorf("the LLM was asked about %q, want %q", got, texts)
	}
	return nil
}

// sameTexts compares as sets: two workers judge in any order.
func sameTexts(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, each := range want {
		seen[each]++
	}
	for _, each := range got {
		seen[each]--
		if seen[each] < 0 {
			return false
		}
	}
	return true
}

// historyStub gives a made-up history cut to the window the way Telegram
// paging does: newest first, down to the cursor, the limit or the days.
type historyStub struct {
	telegram.Client
	history []telegram.Update
}

func (h historyStub) ReadHistory(_ context.Context, _ int64, window telegram.HistoryWindow) (telegram.History, error) {
	var read []telegram.Update
	gap := false
	for each := len(h.history) - 1; each >= 0; each-- {
		message := h.history[each]
		if message.MessageID <= window.After {
			break
		}
		if len(read) == window.Limit || message.Date.Before(window.Since) {
			gap = true
			break
		}
		read = append([]telegram.Update{message}, read...)
	}
	return telegram.History{Messages: read, Gap: gap}, nil
}
