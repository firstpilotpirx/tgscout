## 1. Domain and settings

- [x] 1.1 Red: unit tests for `Cursor` and `MessageTracker` — an in-flight key begins once; out-of-order finish of 10, 11, 12 keeps the cursor at 10 until 11 finishes, then 12; a failed key blocks the cursor and its copies; keys of two chats do not mix
- [x] 1.2 Green: `core/domain` `MessageKey`, `Message`, `QueueItem`, `Decision`, `Judgment`, `Match`, `Cursor`, `MessageTracker`
- [x] 1.3 Red: unit tests for `Settings.RequestsOf` (link and @username of one chat give the same requests, config order) and the parser — `queue_size` default 1000, a given value, below 1 is an error on its line
- [x] 1.4 Green: `Settings.QueueSize`, `RequestsOf`, `queue_size` in `settings_parser.go`; `tgscout.yaml.example`
- [x] 1.5 Commit (`kit ship`)

## 2. Use cases

- [x] 2.1 Red: unit tests for `CollectMessage` with fake ports — empty text dropped; a chat no request watches dropped; an in-flight duplicate dropped; a processed message dropped; a new message queued with every request of its chat; an edit queued with only the requests without a sent «yes»; an edit whose requests all have a sent «yes» dropped
- [x] 2.2 Green: ports `MessageRepository`, `MessageQueue`, `Llm`, `Notifier`; `CollectMessage.Collect`
- [x] 2.3 Red: unit tests for `MatchMessage` — one judgment and one decision per request; a «yes» notified then marked sent; a saved but unsent «yes» judged and sent again; a `Notify` error leaves the message unprocessed and unsent; processed and cursor only after the last request; an LLM error leaves the message unprocessed, the cursor in place and returns the key; an unclear judgment saved as «no» and listed in the outcome; an edit skips a request with a sent «yes» and never moves the cursor; out-of-order finishes move the cursor as the tracker says
- [x] 2.4 Green: `MatchMessage.Match` with the per-key lock; `NewMatchingConfig` wires the tracker
- [x] 2.5 Commit (`kit ship`)

## 3. Secondary adapters

- [x] 3.1 Red: integration tests for `SqliteMessageRepository` on a temp file — save and replace a decision, sent requests (only rows with `sent_at`), mark sent, mark processed, is processed, cursor upsert never moves back
- [x] 3.2 Green: `adapters/secondary/persistence/sqlite/sqlite_message_repository.go`
- [x] 3.3 Red: integration tests for `OpenAiLlmClient` against an `httptest` server — request body (model, system and user messages, temperature 0), yes, no, `Yes.`, `<think>` removed, unclear then yes, unclear twice, 503, timeout
- [x] 3.4 Green: `adapters/secondary/llm/openai/openai_llm_client.go`
- [x] 3.5 Red: integration tests for `ChannelMessageQueue` — put and receive in order; put blocks while full and goes on when a place frees up; both end with `ctx`
- [x] 3.6 Green: `adapters/secondary/queue/channel/channel_message_queue.go`
- [x] 3.7 Red: integration test for `ConsoleNotifier` with a real `MatchPrinter` writing to a buffer
- [x] 3.8 Green: `adapters/secondary/notify/console/console_notifier.go`
- [x] 3.9 Commit (`kit ship`)

## 4. Telegram and primary adapters

- [x] 4.1 Red: unit tests for the gotd update mapping on `tg` values (new and edited messages of a channel and a group, a service message, a caption) and for `ResolveChat`/`Lookup` returning the chat IDs and usernames; update the existing lookup and fake-client tests
- [x] 4.2 Green: `infra/telegram` `Update`, `ChatInfo`, `Client.Listen`; `gotdClient` over gotd's updates manager and dispatcher; the fake client plays the scenario's `updates`
- [x] 4.3 Red: `components/matching/tests/features/slice/message_matching.feature` (`@message_matching @slice`) and its godog runner in `components/matching/tests`; the collector scenarios through the real `CollectMessage` with faked secondary adapters): a service message is dropped, a message with no text is dropped, a photo caption counts as text; an integration test for the collector (a stub `Client` plays updates through `Listen`, mocked use case); unit tests for the link of a public and of a private chat, an update of an unwatched chat, an edit passed as edited, a collect error reported; add godog v0.16.0 to `tgscout/go.mod`
- [x] 4.4 Green: `adapters/primary/collect/telegram`; `.go-arch-lint.yml` lets `adapters` depend on `infra`
- [x] 4.5 Red: the matcher scenario in the same slice feature: a full queue holds new messages back (a real `ChannelMessageQueue` of size 1, one worker, a slow fake `Llm`, the real `MatchMessage`); an integration test for the workers (real queue, mocked use case): an error and a warning reported, workers stop with `ctx`
- [x] 4.6 Green: `adapters/primary/match/channel`
- [ ] 4.7 Commit (`kit ship`)

## 5. CLI

- [ ] 5.1 Red: unit tests for the console lines of the mock — match line and text start (80 characters, line breaks as spaces, `…`), warning with the raw answer, error without request or message text, plain marks
- [ ] 5.2 Green: `apps/cli/console_matching.go` (the console implements `MatchPrinter`)
- [ ] 5.3 Red: update the start unit tests — the workers and the listener start after `connect`, before «running»; the stop ends them before the database closes
- [ ] 5.4 Green: wiring in `apps/cli/start.go` and `start_telegram.go`
- [ ] 5.5 Commit (`kit ship`)

## 6. E2E

- [ ] 6.1 Red: `e2e/features/message_matching.feature` (`@message_matching @e2e`), one `Rule:` per requirement, every E2E scenario of design § Test levels (the slice scenarios run in the adapters' features); the world's fake LM Studio (`httptest`, answers per request text, records calls), the fake Telegram `updates`, reading `tgscout.db` after the stop; the zero queue size row in `e2e/features/settings.feature`
- [ ] 6.2 Green: fix what the scenarios find
- [ ] 6.3 Commit (`kit ship`)

## 7. Docs and version

- [ ] 7.1 `README.md`: what the console shows while running (match, warning, error lines) and the `queue_size` setting
- [ ] 7.2 Bump `VERSION` to 0.4.0
- [ ] 7.3 Commit (`kit ship`)

Manual checks by the user: none for this run (`manual_run` lists no area); the owner's run with LM Studio of the acceptance bar stays for the owner.
