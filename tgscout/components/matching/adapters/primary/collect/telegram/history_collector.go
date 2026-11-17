package telegram

import (
	"context"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// CatchUpUseCase is the catch-up use case.
type CatchUpUseCase interface {
	Open(ctx context.Context, chatID int64, ref domain.ChatRef) (domain.CatchUpChat, error)
	Collect(ctx context.Context, chat domain.CatchUpChat, message domain.Message, rank int) error
	Close(ctx context.Context, chat domain.CatchUpChat) error
	Wait(ctx context.Context) (domain.CatchUpCounts, error)
}

// ChatReport is one chat as catch-up read it; Err is set when the read failed.
type ChatReport struct {
	Chat   Chat
	Read   int
	Window domain.CatchUpChat
	Gap    bool
	Err    error
}

// HasCursor tells whether the chat was read from its cursor.
func (r ChatReport) HasCursor() bool { return r.Window.HasCursor }

// Summary is the end of a catch-up: the counts, the chats read, how many of
// them had a gap and the time it took.
type Summary struct {
	Counts domain.CatchUpCounts
	Chats  int
	Gaps   int
	Took   time.Duration
}

// HistoryReports print what catch-up does.
type HistoryReports struct {
	OnStart  func(chats int)
	OnChat   func(ChatReport)
	OnWait   func(time.Duration)
	OnDone   func(Summary)
	OnFailed func(domain.Message, error)
}

// HistoryCollector reads the missed history of the watched chats, one chat
// after another, and hands each message to the catch-up use case.
type HistoryCollector struct {
	useCase CatchUpUseCase
	reports HistoryReports
	sleep   func(context.Context, time.Duration) error
}

// NewHistoryCollector builds the collector; sleep waits out a FLOOD_WAIT.
func NewHistoryCollector(useCase CatchUpUseCase, reports HistoryReports,
	sleep func(context.Context, time.Duration) error) *HistoryCollector {
	if reports.OnStart == nil {
		reports.OnStart = func(int) {}
	}
	if reports.OnChat == nil {
		reports.OnChat = func(ChatReport) {}
	}
	if reports.OnWait == nil {
		reports.OnWait = func(time.Duration) {}
	}
	if reports.OnDone == nil {
		reports.OnDone = func(Summary) {}
	}
	if reports.OnFailed == nil {
		reports.OnFailed = func(domain.Message, error) {}
	}
	return &HistoryCollector{useCase: useCase, reports: reports, sleep: sleep}
}

// CatchUp reads every chat in the given order, waits until every history
// message is judged, then reports the summary; a stop ends it with no summary.
func (h *HistoryCollector) CatchUp(ctx context.Context, client telegram.Client, chats []Chat) error {
	began := time.Now()
	h.reports.OnStart(len(chats))
	gaps := 0
	for _, chat := range chats {
		report := h.catchUpChat(ctx, client, chat)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if report.Gap {
			gaps++
		}
		h.reports.OnChat(report)
	}
	counts, err := h.useCase.Wait(ctx)
	if err != nil {
		return err
	}
	summary := Summary{Counts: counts, Chats: len(chats), Gaps: gaps, Took: time.Since(began)}
	h.reports.OnDone(summary)
	return nil
}

// catchUpChat opens the chat, reads its window, hands each message on with
// its rank from the newest, and closes it whatever happened.
// The report is a named result, so the deferred Close can still set its error.
func (h *HistoryCollector) catchUpChat(ctx context.Context, client telegram.Client, chat Chat) (report ChatReport) {
	report = ChatReport{Chat: chat}
	window, err := h.useCase.Open(ctx, chat.ID, chat.Ref)
	report.Window = window
	if err != nil {
		report.Err = err
		return report
	}
	defer func() {
		if err := h.useCase.Close(ctx, window); err != nil && report.Err == nil {
			report.Err = err
		}
	}()
	history, err := telegram.Retry(ctx, h.waiting, h.sleep, func() (telegram.History, error) {
		return client.ReadHistory(ctx, chat.ID, telegram.HistoryWindow{After: window.After,
			Limit: window.Depth.Messages, Since: window.Since()})
	})
	if err != nil {
		report.Err = err
		return report
	}
	report.Read = len(history.Messages)
	report.Gap = window.HasCursor && history.Gap
	for i, update := range history.Messages {
		message := toMessage(chat, update)
		if update.Service {
			message.Text = "" // a service message has nothing to judge
		}
		if err := h.useCase.Collect(ctx, window, message, len(history.Messages)-i); err != nil && ctx.Err() == nil {
			h.reports.OnFailed(message, err)
		}
	}
	return report
}

func (h *HistoryCollector) waiting(wait time.Duration) { h.reports.OnWait(wait) }
