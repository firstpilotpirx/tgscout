package main

import (
	"context"
	"database/sql"
	"sync"

	"github.com/firstpilotpirx/tgscout/components/matching"
	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	match "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/match/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/llm/openai"
	notify "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/notify/console"
	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
	queue "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// live is the matching part of a running start: the wired use cases, the
// queue between them and how many workers judge at once.
type live struct {
	config  *matching.MatchingConfig
	queue   match.Receiver
	workers int
}

// newLive wires the matching component over the database and LM Studio.
func newLive(settings application.SettingsRepository, db *sql.DB, con *console) live {
	resolved := settings.Settings()
	messages := queue.NewChannelMessageQueue(resolved.QueueSize)
	config := matching.NewMatchingConfig(matching.Ports{
		Settings: settings,
		Messages: sqlite.NewSqliteMessageRepository(db),
		Queue:    messages,
		Llm:      openai.NewOpenAiLlmClient(resolved.LmStudio),
		Notifier: notify.NewConsoleNotifier(con),
	})
	return live{config: config, queue: messages, workers: resolved.Workers}
}

// serve starts the workers, says it is running, then listens to the found
// chats until ctx ends; it returns once the listener and every worker
// stopped. A listener that fails stops the workers and gives its error.
func (l live) serve(ctx context.Context, con *console, client telegram.Client,
	found map[string]telegram.ChatInfo) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		match.RunWorkers(ctx, l.workers, l.queue, l.config.MatchMessage, con)
	}()
	con.running()
	collector := collect.NewTelegramCollector(watchedByID(found), l.config.CollectMessage, con.collectFailed)
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
		chats[info.ID] = collect.Chat{Ref: ref, Username: info.Username}
	}
	return chats
}
