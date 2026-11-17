package main

import (
	"context"
	"database/sql"
	"sync"
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
	queue    match.Receiver
	workers  int
	sleep    func(context.Context, time.Duration) error
}

// newLive wires the matching adapters over the database and LM Studio.
func newLive(settings application.SettingsRepository, db *sql.DB,
	sleep func(context.Context, time.Duration) error) live {
	resolved := settings.Settings()
	messages := queue.NewChannelMessageQueue(resolved.QueueSize)
	return live{settings: resolved, queue: messages, workers: resolved.Workers, sleep: sleep,
		ports: matching.Ports{
			Settings: settings,
			Messages: sqlite.NewSqliteMessageRepository(db),
			Queue:    messages,
			Llm:      openai.NewOpenAiLlmClient(resolved.LmStudio),
		}}
}

// wire adds the Telegram notifier over the connected client and builds the
// use cases.
func (l live) wire(con *console, client telegram.Client) *matching.MatchingConfig {
	ports := l.ports
	ports.Notifier = notify.NewTelegramNotifierClient(client, l.settings.Owner.String(),
		notify.PrintFunc(con.PrintMatch), con.sendFloodWait, l.sleep)
	return matching.NewMatchingConfig(ports)
}

// serve starts the workers, says it is running, then catches up the found
// chats in the background and listens to them until ctx ends; it returns once the listener and every worker
// stopped. A listener that fails stops the workers and gives its error.
func (l live) serve(ctx context.Context, con *console, client telegram.Client,
	found map[string]telegram.ChatInfo) error {
	config := l.wire(con, client)
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		match.RunWorkers(ctx, l.workers, l.queue, config.MatchMessage,
			match.ReportFuncs{OnUnclear: con.Unclear, OnFailed: con.Failed})
	}()
	con.running()
	wait.Add(1)
	go func() {
		defer wait.Done()
		history := collect.NewHistoryCollector(config.CatchUp, collect.HistoryReports{
			OnStart: con.catchUpStarted, OnChat: con.catchUpChat, OnWait: con.floodWait,
			OnDone: con.catchUpDone, OnFailed: con.collectFailed,
		}, l.sleep)
		_ = history.CatchUp(ctx, client, inConfigOrder(l.settings, found)) // it ends with an error only on a stop
	}()
	collector := collect.NewTelegramCollector(watchedByID(found), config.CollectMessage, con.collectFailed)
	err := collector.Listen(ctx, client)
	stop()
	wait.Wait()
	if isStop(err) {
		return nil
	}
	return err
}

// watchedByID maps each found chat's peer ID to its config reference.
func watchedByID(found map[string]telegram.ChatInfo) map[int64]collect.Chat {
	chats := make(map[int64]collect.Chat, len(found))
	for text, info := range found {
		ref, err := domain.NewChatRef(text)
		if err != nil {
			continue // the config check let only valid references through
		}
		chats[info.ID] = collect.Chat{ID: info.ID, Ref: ref, Username: info.Username, Title: info.Title}
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
