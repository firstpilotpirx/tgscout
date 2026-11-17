## Context

`walking-skeleton`, `hierarchical-settings` and `telegram-sign-in` built the start: config → SQLite (tables `chat_cursor`, `processed_message`, `decision`) → sign-in → lookup of the chats and the owner → «running» until Ctrl+C. The matching component already has the empty use cases `CollectMessage`, `MatchMessage` and the empty ports `MessageRepository`, `MessageQueue`, `Llm`, `Notifier`. `infra/telegram` holds the `Client` port over gotd v0.163.0, built with `NoUpdates: true`, and a fake client under the build tag `telegramfake` that the E2E build reads from a JSON scenario.

This slice fills the message path of the [message-path flow](../../../docs/flows/message-path.html) and the [L4 matching overview](../../../docs/modules/matching/overview.html). The preparation recorded: edits judged again only where the decision was «no»; an unclear answer asked once more, then «no» with a warning; rules in the system message, request and message in the user message; a root setting `queue_size` (default 1000, at least 1) with a waiting collector; a duplicate is the same chat and message ID.

## Goals / Non-Goals

**Goals:** new and edited messages of the watched chats reach `CollectMessage`; the pre-filter; the queue with back-pressure; `workers` matcher workers; the LLM client over the OpenAI-compatible API; decisions, processed marks and the cursor in SQLite; the console notifier; the `queue_size` setting.

**Non-Goals:** the private message to the owner (`owner-notification`), history at start (`catch-up`), retries, backoff, graceful drain and reconnects (`delivery-resilience`), latency and load numbers (`quality-targets`).

## Decisions

### 1. Domain values

`core/domain` gets:

- `MessageKey{ChatID int64, MessageID int}` — the identity of a message; a duplicate is the same key.
- `Message{Key, Chat ChatRef, SentAt time.Time, Text, Link string, Edited bool}` — what the collector hands in; `Chat` is the reference as the config writes it.
- `QueueItem{Message, Requests []Request}` — the message with the requests of its chat, so the matcher never reads the config.
- `Decision{Key, RequestID string, Matched bool, DecidedAt time.Time}`.
- `Judgment{Matched, Unclear bool, Raw string}` — what the LLM port returns.
- `Cursor` — the cursor of one chat in memory: the begun and the finished message IDs and the position; `Finish(id)` moves the position to the highest finished ID below which no begun ID is unfinished.
- `MessageTracker` — the in-memory state both use cases share, guarded by a mutex: `Begin(key) bool` (false when the key is in flight already: the in-run duplicate), `Finish(key) (cursor int, moved bool)`, `Fail(key)`; it holds one `Cursor` per chat, where the cursor rule lives. A failed key stays in flight for the rest of the run, so the cursor never passes it and its copies are dropped; retries come with `delivery-resilience`. Edits never begin or finish a key.
- `Settings.QueueSize int`; `Settings.RequestsOf(chat ChatRef) []Request` — every request that watches the chat (by `ChatRef.Key()`), in config order.

### 2. Ports

```go
type MessageRepository interface {
    IsProcessed(ctx, MessageKey) (bool, error)
    SentRequests(ctx, MessageKey) ([]string, error)      // request IDs whose «yes» was sent (sent_at set)
    SaveDecision(ctx, Decision) error                     // insert or replace the row of (chat, message, request)
    MarkSent(ctx, MessageKey, requestID string, at time.Time) error
    MarkProcessed(ctx, MessageKey, at time.Time) error
    MoveCursor(ctx, chatID int64, messageID int, at time.Time) error // never moves it back
}
type MessageQueue interface { Put(ctx, QueueItem) error }  // blocks while full
type Llm interface { Judge(ctx, find, text string) (Judgment, error) }
type Notifier interface { Notify(ctx, Match) error }       // Match{Message, Request}
```

The worker side of the queue is not a port: the primary adapter `primary/match/channel` receives from `ChannelMessageQueue.Receive(ctx)` (the dashed «receives» link of the L4 page).

### 3. Use cases

`CollectMessage.Collect(ctx, Message) error`:

1. No requests watch the chat, or the text is empty after trimming → drop. (Service messages never reach it: the collector adapter skips them, §5.)
2. A new message: `tracker.Begin(key)` false → drop (in flight); `IsProcessed` true → `tracker.Finish(key)` and drop (judged in an earlier run). Otherwise `Put` the item with every request of the chat.
3. An edit: requests of the chat minus `SentRequests(key)`; none left → drop; else `Put` the item with those requests and `Edited: true`.

`MatchMessage.Match(ctx, QueueItem) (Outcome, error)`: holds a lock per message key for the whole item (an edit and the original never interleave); per request it skips one that `SentRequests` already lists (so a sent «yes» is never sent twice), calls `Judge`, saves the decision, and on a «yes» calls `Notify` then `MarkSent`. A «yes» saved but never sent (a crash or a `Notify` error in between) has no `sent_at`, so the next judgment of that message asks again and sends again: delivery stays at least once (`docs/design-progress.md`: «no match is ever lost, a rare repeat after a crash is allowed»). An `Llm` or `Notify` error stops the item: no `MarkSent`, no processed mark, no cursor move, `tracker.Fail(key)` (new messages only) and the error is returned with the key. After the last request of a new message: `MarkProcessed`, then `tracker.Finish(key)`, and `MoveCursor` when it moved. An edit of a message that was never processed (edited before tgscout saw it) is judged and marked processed like a new one but never moves the cursor. `Outcome` lists the unclear judgments, which the worker prints as warnings.

### 4. Secondary adapters

- `secondary/queue/channel` — `ChannelMessageQueue`, a buffered channel of `queue_size`; `Put` blocks until a place frees up or `ctx` ends; `Receive(ctx)` returns the next item or `ctx`'s error.
- `secondary/llm/openai` — `OpenAiLlmClient` over `net/http` (no SDK: one POST): `POST <url>/chat/completions` with `{model, temperature: 0, max_tokens: 256, messages: [system, user]}` (room for a reasoning model's `<think>` block); the system message holds the fixed rules («answer only yes or no»), the user message holds `Request: <find>` and `Message: <text>`; the `http.Client` timeout is `lm_studio.timeout`. The answer's first choice is trimmed, a `<think>…</think>` block removed, lowercased and stripped of punctuation: `yes` / `no`; anything else is unclear and is asked once more with a stricter system message («Reply with exactly one word: yes or no»); a second unclear answer returns `Judgment{Unclear: true, Raw}`, which `MatchMessage` saves as «no». A non-2xx status, a broken body or a transport error is an error that names the status or the cause, never the prompt.
- `secondary/notify/console` — `ConsoleNotifier`, the notifier of this slice (the L3 page's notifier writes «private message + console»; `owner-notification` adds the Telegram part): it hands each match to a `MatchPrinter` interface that `apps/cli`'s console implements, since only the app may use the color library.
- `secondary/persistence/sqlite` — `SqliteMessageRepository` on the existing tables: `SaveDecision` is `INSERT … ON CONFLICT (chat_id, message_id, request_id) DO UPDATE`, `MoveCursor` is an upsert with `max(last_message_id, ?)`. No new migration.

### 5. Primary adapters and Telegram

- `infra/telegram` gets `Update{ChatID int64, MessageID int, Date time.Time, Text string, Service, Edited bool}` and the client method `Listen(ctx, func(context.Context, Update)) error`, which blocks until `ctx` ends. `gotdClient` swaps `NoUpdates: true` for gotd's `updates.Manager` (`telegram/updates`, in-memory state) over a `tg.UpdateDispatcher` with `OnNewMessage`, `OnNewChannelMessage`, `OnEditMessage`, `OnEditChannelMessage`; a `tg.MessageService` is `Service`; a `tg.Message`'s `Message` field is the text or the caption. Chat IDs are the bare peer IDs (`PeerChannel.ChannelID`, `PeerChat.ChatID`).
- `ResolveChat` returns `ChatInfo{State, ID int64, Username string}` instead of the bare state, so the start knows which peer ID each config chat has; `Lookup.Find` returns the found chats by their config text. A `-100…` or `-…` numeric reference maps to the bare ID it already compares with.
- The fake client plays a list of made-up `updates` from its JSON scenario after `Listen` starts, each `{chat, id, text, service, edited, after_ms}`; `chat` is the config text of a watched chat, and the fake gives each chat a made-up ID.
- `primary/collect/telegram` — the collector: built with the found chats (`peer ID → ChatRef, username`), it maps each `Update` of a watched chat to a `domain.Message`: service updates are dropped here, the link is `https://t.me/<username>/<id>` for a chat with a username, `https://t.me/c/<id>/<message>` otherwise. It calls `CollectMessage`; an error (the repository) is printed as an error line through a report function `apps/cli` passes in.
- `primary/match/channel` — `workers` goroutines, each `Receive` → `MatchMessage.Match` → report the outcome or the error; they stop when `ctx` ends.
- `.go-arch-lint.yml`: `adapters` may depend on `infra` (the L3 page: `infra/telegram` is «shared by collect and notify»; `architecture.md`: component → `infra/*`). This replaces the `walking-skeleton` decision «Only `apps/cli` imports adapters and `infra`»: `apps/cli` still is the only place that imports adapters.

### 6. CLI wiring and console

`apps/cli/start.go` builds the repository, the queue, the LLM client, the console notifier and the tracker, passes them to `matching.NewMatchingConfig`, and inside `client.Run` after `connect` starts the workers and `client.Listen` with the collector, then prints «running». The stop ends the context: the listener and the workers stop, then the database closes as today. The console of `apps/cli` implements `MatchPrinter` and gets `match`, `matchWarning` and `matchError` lines as in the [mock](mock/index.html): text start at most 80 characters with line breaks as spaces and `…` when cut, the raw unclear answer at most 80 characters, errors without the request or message text.

### 7. `queue_size`

`settings_parser.go` reads the root key `queue_size` like `workers` (a number, at least 1, default `DefaultQueueSize = 1000`); `tgscout.yaml.example` documents it.

## Test levels

| Scenario | Level | Where |
|---|---|---|
| one message, two requests of its chat | E2E | `e2e/features/message_matching.feature` |
| a service message is dropped | slice | `components/matching/tests/features/slice/message_matching.feature` (`@message_matching @slice`, one godog runner in `components/matching/tests`): the collector through the real `CollectMessage` with faked secondary adapters |
| a message with no text is dropped | slice | collector feature |
| a photo caption counts as text | slice | collector feature |
| a processed message is skipped | E2E | feature (the fake sends the same message twice; a second start with the same database) |
| the same text in another message is judged again | E2E | feature |
| the LLM is asked with the request and the message | E2E | feature (the fake LM Studio records each call) |
| an unclear answer is asked again, then saved as no | E2E | feature |
| a matching message is shown on the console | E2E | feature (the SQLite file is read after the stop) |
| a message that does not match | E2E | feature |
| the cursor never passes an unprocessed message | unit | `core/domain` `MessageTracker` test and `MatchMessage` test with an out-of-order finish: the order of two workers cannot be forced from outside the binary |
| LM Studio is down | E2E | feature (the fake LM Studio answers 503) |
| an edited message is judged again only where it was «no» | E2E | feature |
| a match saved but not shown is shown again | unit | `MatchMessage` test: a fake repository holds a «yes» without `sent_at`; the crash between save and send cannot be forced from outside the binary |
| a full queue holds new messages back | slice | `components/matching/tests/features/slice/message_matching.feature` (`@message_matching @slice`, one godog runner in `components/matching/tests`): a real `ChannelMessageQueue` of size 1, a slow fake `Llm`, one worker, through the real `MatchMessage` |
| settings: defaults fill what the file leaves out | unit | `settings_parser_test.go` (queue size 1000) |
| settings: an invalid config stops the start (zero queue size) | E2E | `e2e/features/settings.feature`, a new Examples row |
| settings: every form of chat reference is accepted | E2E | `e2e/features/settings.feature`, unchanged |
| settings: every problem is listed at once | E2E | `e2e/features/settings.feature`, unchanged |

The E2E world starts a fake OpenAI-compatible server (`httptest`, in the `e2e` module) per scenario that answers per request text (`yes`, `no`, an unclear text, or 503) and records every call; the config's `lm_studio.url` points at it. The fake Telegram scenario gets the `updates`. A scenario waits for the expected console lines, then stops tgscout with SIGINT and reads `tgscout.db` with `mattn/go-sqlite3` (already in `e2e/go.mod`).

Integration tests (`-tags=integration`, one `_integration_test.go` per adapter, `testing-integration-go.md`): `ChannelMessageQueue` (real channel across goroutines: order, blocking while full, `ctx`), `ConsoleNotifier` (a real `MatchPrinter` writing to a buffer), the collector (a stub `Client` that plays updates through a real `Listen` call, mocked use case), the matcher workers (real queue, mocked use case), `SqliteMessageRepository` on a temp file (upsert, max cursor, sent requests) and `OpenAiLlmClient` against an `httptest` server (body shape, yes, no, `<think>` removed, unclear twice, 503, timeout). Unit tests: the use cases with fake ports, `MessageTracker`, the collector's update mapping and links, the console lines, the parser.

This keeps the override `telegram-sign-in` named: the gotd network path (`Listen` over the updates manager) gets no automated test against real Telegram; the owner's manual run with LM Studio covers it (proposal § Acceptance bar). The fake LM Studio in E2E overrides `testing-e2e-go.md` «External deps: Real or production-equivalent. No httptest» for the LLM only, as the proposal's § Test override asks; the owner approves it with the mock before any code (no recorded preparation decision covers it). The request goes over real HTTP to a made-up server; the owner's manual run with LM Studio covers the real model.

Every primary adapter gets its slice `.feature` (`testing.md` R-073): godog (`github.com/cucumber/godog` v0.16.0, already in `e2e/go.mod`) joins the `tgscout` module as a test dependency, run from `go test` in `components/matching/tests` over the one slice feature of the capability (`gherkin.md` R-064, `testing.md` R-073).

## Risks / Trade-offs

- [gotd's updates manager is not covered by automated tests] → thin mapping, unit-tested on `tg` values; the manual run covers the network.
- [A failed message blocks the cursor of its chat for the rest of the run] → intended («the cursor does not move past it»); `delivery-resilience` adds the retry.
- [The tracker keeps every failed key in memory] → only failures stay; finished keys are dropped once the cursor passes them.
- [Small local models answer with extra words] → `max_tokens: 256`, normalization, one stricter retry, then a warning.

## Interface mock

[Live matching mock](mock/index.html) — match, warning and error lines in dark, light and plain output.

## Release

MINOR, 0.3.0 → 0.4.0, bumped in the last task group.
