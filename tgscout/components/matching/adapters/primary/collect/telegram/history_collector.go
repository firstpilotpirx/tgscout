package telegram

import (
	"context"
	"errors"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// CatchUpUseCase is the catch-up use case.
type CatchUpUseCase interface {
	Open(ctx context.Context, chatID int64, ref domain.ChatRef) (domain.CatchUpChat, error)
	Collect(ctx context.Context, chat domain.CatchUpChat, message domain.Message, rank int) (bool, error)
	Close(ctx context.Context, chat domain.CatchUpChat) error
	Pause(chatIDs []int64)
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

// HistoryReports print what catch-up does. OnRefreshed tells that a stored
// chat Telegram no longer knew was looked up again; gapRead is true when the
// gap read after a reconnect found it.
type HistoryReports struct {
	OnStart     func(chats int)
	OnChat      func(ChatReport)
	OnWait      func(time.Duration)
	OnDone      func(Summary)
	OnFailed    func(domain.Message, error)
	OnRefreshed func(chat Chat, gapRead bool)
}

// Refresher looks a stored chat up by name again once Telegram no longer
// knows it; it gives telegram.ErrChatGone when the chat cannot be read.
type Refresher interface {
	Refresh(ctx context.Context, ref string, id int64) (telegram.ChatInfo, error)
}

// HistoryCollector reads the missed history of the watched chats, one chat
// after another, and hands each message to the catch-up use case.
type HistoryCollector struct {
	useCase   CatchUpUseCase
	reports   HistoryReports
	sleep     func(context.Context, time.Duration) error
	refresher Refresher
}

// WithRefresher makes the collector look a gone chat up again through r; with
// none a gone chat fails its read as any failed read does.
func (h *HistoryCollector) WithRefresher(r Refresher) *HistoryCollector {
	h.refresher = r
	return h
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
	if reports.OnRefreshed == nil {
		reports.OnRefreshed = func(Chat, bool) {}
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
		report, _ := h.readChat(ctx, client, chat, false)
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

// GapRead reads every chat again from its cursor after the connection to
// Telegram was back, the same way catch-up does, and gives how many messages
// it handed to the judge and the chats whose read failed.
func (h *HistoryCollector) GapRead(ctx context.Context, client telegram.Client, chats []Chat) (int, []ChatReport) {
	handed := 0
	var failed []ChatReport
	for _, chat := range chats {
		report, n := h.readChat(ctx, client, chat, true)
		if ctx.Err() != nil {
			return handed, failed
		}
		handed += n
		if report.Err != nil {
			failed = append(failed, report)
		}
	}
	return handed, failed
}

// readChat opens the chat, reads its window, hands each message on with its
// rank from the newest, and gives the report and how many went to the judge.
// It closes the chat (releases its cursor) only after a read that worked: a
// failed read keeps the cursor held, so later messages never move it past the
// unread history. gapRead tells the reports which read found a gone chat.
func (h *HistoryCollector) readChat(ctx context.Context, client telegram.Client, chat Chat,
	gapRead bool) (ChatReport, int) {
	report := ChatReport{Chat: chat}
	window, err := h.useCase.Open(ctx, chat.ID, chat.Ref)
	report.Window = window
	if err != nil {
		report.Err = err
		return report, 0
	}
	history, err := h.readWindow(ctx, client, chat, window, gapRead)
	if err != nil {
		report.Err = err
		return report, 0
	}
	report.Read = len(history.Messages)
	report.Gap = window.HasCursor && history.Gap
	handed := 0
	for i, update := range history.Messages {
		message := toMessage(chat, update)
		if update.Service {
			message.Text = "" // a service message has nothing to judge
		}
		queued, err := h.useCase.Collect(ctx, window, message, len(history.Messages)-i)
		if err != nil && ctx.Err() == nil {
			h.reports.OnFailed(message, err)
		}
		if queued {
			handed++
		}
	}
	if err := h.useCase.Close(ctx, window); err != nil {
		report.Err = err
	}
	return report, handed
}

// readWindow reads the chat's window; when Telegram no longer knows a stored
// chat, it looks the chat up again once and reads once more.
func (h *HistoryCollector) readWindow(ctx context.Context, client telegram.Client, chat Chat,
	window domain.CatchUpChat, gapRead bool) (telegram.History, error) {
	read := func() (telegram.History, error) {
		return telegram.Retry(ctx, h.waiting, h.sleep, func() (telegram.History, error) {
			return client.ReadHistory(ctx, chat.ID, telegram.HistoryWindow{After: window.After,
				Limit: window.Depth.Messages, Since: window.Since()})
		})
	}
	history, err := read()
	if errors.Is(err, telegram.ErrChatGone) && h.refresher != nil {
		if _, refreshErr := h.refresher.Refresh(ctx, chat.Ref.String(), chat.ID); refreshErr == nil {
			h.reports.OnRefreshed(chat, gapRead)
			history, err = read()
		}
	}
	return history, err
}

func (h *HistoryCollector) waiting(wait time.Duration) { h.reports.OnWait(wait) }
