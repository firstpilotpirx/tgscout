package telegram

import (
	"context"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// ReconnectReports print what the reconnect does.
type ReconnectReports struct {
	OnLost    func()
	OnBack    func(chats int)
	OnGapDone func(handed int, failed []ChatReport)
}

// Pauser holds the cursor of each chat while the connection is away.
type Pauser interface {
	Pause(chatIDs []int64)
}

// Reconnector follows the connection to Telegram: on a drop it holds every
// watched chat's cursor, and once the connection is back it reads each chat
// again from its cursor. A return while a gap read runs makes one more read
// after it.
type Reconnector struct {
	pauser  Pauser
	history *HistoryCollector
	client  telegram.Client
	chats   []Chat
	reports ReconnectReports
	events  chan telegram.ConnectionEvent
}

// NewReconnector builds the reconnector over the watched chats.
func NewReconnector(pauser Pauser, history *HistoryCollector, client telegram.Client, chats []Chat,
	reports ReconnectReports) *Reconnector {
	if reports.OnLost == nil {
		reports.OnLost = func() {}
	}
	if reports.OnBack == nil {
		reports.OnBack = func(int) {}
	}
	if reports.OnGapDone == nil {
		reports.OnGapDone = func(int, []ChatReport) {}
	}
	return &Reconnector{pauser: pauser, history: history, client: client, chats: chats, reports: reports,
		events: make(chan telegram.ConnectionEvent, 64)}
}

// Handle takes one connection event; it never waits.
func (r *Reconnector) Handle(event telegram.ConnectionEvent) {
	select {
	case r.events <- event:
	default: // far behind: the queued events already lead to a gap read
	}
}

// Run follows the events until ctx ends.
func (r *Reconnector) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-r.events:
			if r.take(event) {
				r.gapReads(ctx)
			}
		}
	}
}

// take applies one event and says whether the connection is back.
func (r *Reconnector) take(event telegram.ConnectionEvent) bool {
	if event == telegram.ConnectionLost {
		r.pauser.Pause(r.chatIDs())
		r.reports.OnLost()
		return false
	}
	return event == telegram.ConnectionBack
}

// gapReads reads the gap, then once more while a return came during the read.
func (r *Reconnector) gapReads(ctx context.Context) {
	for again := true; again && ctx.Err() == nil; {
		r.reports.OnBack(len(r.chats))
		handed, failed := r.history.GapRead(ctx, r.client, r.chats)
		if ctx.Err() != nil {
			return
		}
		r.reports.OnGapDone(handed, failed)
		again = r.drain()
	}
}

// drain applies the events that came during a read and says whether one was a return.
func (r *Reconnector) drain() bool {
	back := false
	for {
		select {
		case event := <-r.events:
			back = r.take(event) || back
		default:
			return back
		}
	}
}

func (r *Reconnector) chatIDs() []int64 {
	ids := make([]int64, len(r.chats))
	for i, chat := range r.chats {
		ids[i] = chat.ID
	}
	return ids
}
