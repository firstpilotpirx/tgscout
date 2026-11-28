package main

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching"
	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	match "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/match/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/llm/openai"
	notify "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/notify/telegram"
	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
	queue "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// live is the matching part of a running start: the resolved settings, the
// driven adapters but the notifier, which needs the connected client, the
// queue between the use cases and how many workers judge at once.
type live struct {
	settings domain.Settings
	ports    matching.Ports
	queue    *queue.ChannelMessageQueue
	workers  int
	sleep    func(context.Context, time.Duration) error
	stopping *atomic.Bool // serve printed the stopping line
	stats    *domain.RunStats
	// refresher looks a stored chat Telegram no longer knows up again; nil fails its read.
	refresher collect.Refresher
	// known keeps the found chats and the owner for the next start; nil keeps nothing.
	known telegram.KnownPeers
}

// newLive wires the matching adapters over the database and LM Studio.
func newLive(settings application.SettingsRepository, db *sql.DB,
	sleep func(context.Context, time.Duration) error) live {
	resolved := settings.Settings()
	messages := queue.NewChannelMessageQueue(resolved.QueueSize)
	stats := domain.NewRunStats(time.Now())
	return live{settings: resolved, queue: messages, workers: resolved.Workers, sleep: sleep, stopping: &atomic.Bool{},
		stats: stats,
		ports: matching.Ports{
			Stats:    stats,
			Settings: settings,
			Messages: sqlite.NewSqliteMessageRepository(db),
			Queue:    messages,
			Llm:      openai.NewOpenAiLlmClient(resolved.LmStudio),
		}}
}

// wire adds the Telegram notifier over the connected client and the retry
// rule, and builds the use cases.
func (l live) wire(con *console, client telegram.Client, delivery application.Delivery) *matching.MatchingConfig {
	ports := l.ports
	ports.Notifier = notify.NewTelegramNotifierClient(client, l.settings.Owner.String(),
		notify.PrintFunc(con.PrintMatch), con.sendFloodWait, l.sleep)
	ports.Delivery = delivery
	return matching.NewMatchingConfig(ports)
}

// delivery is the retry rule of the run: its pauses end at the stop (design D4).
func (l live) delivery(con *console, stop context.Context) application.Delivery {
	return application.Delivery{
		Backoff: domain.Backoff{Max: l.settings.RetryMaxPause},
		Sleep: func(ctx context.Context, pause time.Duration) error {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			defer context.AfterFunc(stop, cancel)()
			return application.SleepContext(ctx, pause)
		},
		Reporter: application.DeliveryFuncs{
			OnLlmDown:    con.llmDown,
			OnLlmBack:    func() { con.llmBack(l.queue.Len()) },
			OnGaveUp:     con.gaveUp,
			OnSendFailed: con.sendFailed,
		},
	}
}

// serve starts the workers, says it is running, then catches up the found
// chats in the background and listens to them until ctx ends; it returns once
// the listener and every worker stopped. At the stop the workers take no more
// messages and get stop_grace to finish the ones they hold (design D7). A
// listener that fails stops the workers at once and gives its error.
func (l live) serve(ctx context.Context, con *console, client telegram.Client,
	found map[string]telegram.ChatInfo) error {
	stopCtx, stop := context.WithCancel(ctx)
	defer stop()
	busy := &match.Busy{}
	workCtx, endWork, announced := l.stopInTwoSteps(ctx, con, busy)
	defer announced()
	defer endWork()
	l.queue.OnFull = con.queueFull
	config := l.wire(con, client, l.delivery(con, stopCtx))
	var wait sync.WaitGroup
	wait.Go(func() {
		match.RunWorkers(stopCtx, workCtx, match.Pool{Workers: l.workers, Queue: l.queue, Matcher: config.MatchMessage,
			Report: match.ReportFuncs{OnUnclear: con.Unclear, OnFailed: con.Failed}, Busy: busy})
	})
	con.running()
	wait.Go(func() { l.printStatus(stopCtx, con) })
	readers := l.readHistory(stopCtx, con, client, config, inConfigOrder(l.settings, found))
	collector := collect.NewTelegramCollector(watchedByID(found), config.CollectMessage, con.collectFailed)
	err := collector.Listen(stopCtx, client)
	if ctx.Err() == nil {
		endWork() // the listener failed: no grace time
	}
	stop()
	readers.Wait()
	wait.Wait()
	if isStop(err) {
		return nil
	}
	return err
}

// printStatus prints the status line every status_interval until the stop
// (design D4).
func (l live) printStatus(stop context.Context, con *console) {
	if l.settings.StatusInterval <= 0 {
		return
	}
	ticker := time.NewTicker(l.settings.StatusInterval)
	defer ticker.Stop()
	con.runStatus(stop, ticker.C, func(now time.Time) domain.Status { return l.stats.Cut(l.queue.Len(), now) })
}

// total is the status since start.
func (l live) total() domain.Status { return l.stats.Total(l.queue.Len(), time.Now()) }

// stopInTwoSteps gives the work context of the workers: it ends stop_grace
// after ctx ends, once the stopping line is out. announced waits for that
// line when the stop came.
func (l live) stopInTwoSteps(ctx context.Context, con *console, busy *match.Busy) (context.Context, context.CancelFunc, func()) {
	workCtx, endWork := context.WithCancel(context.WithoutCancel(ctx))
	printed := make(chan struct{})
	notYet := context.AfterFunc(ctx, func() {
		defer close(printed)
		if l.stopping != nil {
			l.stopping.Store(true)
		}
		con.stopping(busy.Count(), l.settings.StopGrace)
		time.AfterFunc(l.settings.StopGrace, endWork)
	})
	announced := func() {
		if !notYet() {
			<-printed // the stopping line is out before serve returns
		}
	}
	return workCtx, endWork, announced
}

// readHistory runs the catch-up of the chats and the gap reads after a
// reconnect in the background, until the stop; wait on the group it gives.
func (l live) readHistory(stop context.Context, con *console, client telegram.Client,
	config *matching.MatchingConfig, chats []collect.Chat) *sync.WaitGroup {
	history := collect.NewHistoryCollector(config.CatchUp, collect.HistoryReports{
		OnStart: func(chats int) { con.catchUpStarted(chats, l.settings.HistoryPause) }, OnChat: con.catchUpChat, OnWait: con.floodWait,
		OnDone: con.catchUpDone, OnFailed: con.collectFailed, OnRefreshed: con.chatRefreshed,
	}, l.sleep)
	if l.refresher != nil {
		history.WithRefresher(l.refresher)
	}
	reconnector := collect.NewReconnector(config.CatchUp, history, client, chats, collect.ReconnectReports{
		OnLost: con.connectionLost, OnBack: con.connectionBack, OnGapDone: con.gapRead,
	})
	client.WatchConnection(reconnector.Handle)
	wait := &sync.WaitGroup{}
	wait.Go(func() {
		_ = history.CatchUp(stop, client, chats) // it ends with an error only on a stop
	})
	wait.Go(func() { reconnector.Run(stop) })
	return wait
}

// announcedStop tells whether serve printed the stopping line.
func (l live) announcedStop() bool { return l.stopping != nil && l.stopping.Load() }

// queued is how many messages wait in the queue.
func (l live) queued() int { return l.queue.Len() }

// watchedByID maps each found chat's peer ID to its config reference.
func watchedByID(found map[string]telegram.ChatInfo) map[int64]collect.Chat {
	chats := make(map[int64]collect.Chat, len(found))
	for text, info := range found {
		ref, err := domain.NewChatRef(text)
		if err != nil {
			continue // the config check let only valid references through
		}
		chats[info.ID] = collect.Chat{ID: info.ID, Ref: ref, Username: info.Username, Title: info.Title,
			Basic: info.Basic}
	}
	return chats
}

// inConfigOrder lists the found chats in the order the config names them.
func inConfigOrder(settings domain.Settings, found map[string]telegram.ChatInfo) []collect.Chat {
	byID := watchedByID(found)
	var chats []collect.Chat
	for _, text := range watchedChats(settings) {
		if info, ok := found[text]; ok {
			chats = append(chats, byID[info.ID])
		}
	}
	return chats
}
