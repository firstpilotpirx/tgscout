# Tasks

## 1. Domain and settings

- [x] 1.1 Red: unit tests for `domain.Backoff.Pause` (1, 2, 4 s, capped at `Max`, no overflow at a large try); the settings parser (`retry_max_pause` and `stop_grace` read, defaults 60 s and 10 s, a value that is not a positive duration refused with field and line)
- [x] 1.2 Green: `core/domain/backoff.go`, `Settings.RetryMaxPause` and `StopGrace`, the YAML parser; `tgscout.yaml.example` names both fields
- [x] 1.3 Commit (`kit ship`)

## 2. Retries in the match use case

- [x] 2.1 Red: unit tests for `MatchMessage` with fake ports and a recording sleep: pauses 1, 2, 4, 4 s with a 4 s cap; a rejecting LLM saved as «no» after 10 tries, processed, `GaveUp` reported; 12 no-answers then «yes» sent, no «no» saved; `LlmDown` once and `LlmBack` once over two messages; a send that fails twice then works (two `SendFailed`, one send, processed after it); a pause that ends at the stop leaves the message unprocessed and the cursor unmoved; a failed database write still settles the catch-up tally as an error; `OpenAiLlmClient` wraps `ErrLlmRejected` for a 400 and an unreadable answer, not for a 503 or no connection
- [x] 2.2 Green: `core/application/delivery.go` (`Delivery`, `DeliveryReporter`, `ErrLlmRejected`, `llmHealth`), `match_message.go`, `matching_config.go` (passes `Delivery`), `adapters/secondary/llm/openai` (rejected errors)
- [x] 2.3 Red: slice scenario «a send fails, then works» in `tests/features/slice/owner_notification.feature` over the stub client
- [x] 2.4 Green: fix what the slice finds; the slice harness passes a `Delivery`
- [x] 2.5 Commit (`kit ship`)

## 3. Queue and workers

- [x] 3.1 Red: unit tests for `ChannelMessageQueue` (`OnFull` once per full episode, again after the queue was empty, `Len`) and `RunWorkers` (`Receive` ends at the stop context, a `Match` in work runs on the work context and ends at it, the busy count)
- [x] 3.2 Green: `adapters/secondary/queue/channel`, `adapters/primary/match/channel/match_workers.go`
- [x] 3.3 Commit (`kit ship`)

## 4. Telegram connection events

- [x] 4.1 Red: unit tests for the gotd connection-state mapping (first Ready no event, Disconnected gives Lost, the next Ready gives Back, the callback never blocks); the fake client plays `connection_lost`, `connection_back`, a `lost` update (not delivered, served by `ReadHistory` after it is played) and `send_hang_ms`; every stub of `Client` gets `WatchConnection`
- [x] 4.2 Green: `infra/telegram` `ConnectionEvent`, `WatchConnection` in the port, the gotd client (`OnConnectionState`), the fake client
- [x] 4.3 Commit (`kit ship`)

## 5. Reconnect and gap read

- [x] 5.1 Red: unit tests for `CatchUp.Pause` (holds each chat's cursor; a finish while held does not move it) and `CatchUp.Collect` returning whether the message was handed on; `HistoryCollector` test: a failed read keeps the chat's cursor held; `GapRead` reads each chat from its cursor and reports the handed count; `Reconnector` (Lost pauses and reports, Back runs one gap read, a Back during a read runs one more); slice scenario «a message processed during the drop is not judged twice» in `tests/features/slice/catch_up.feature`
- [x] 5.2 Green: `core/application/catch_up.go`, `adapters/primary/collect/telegram/history_collector.go` and `reconnect.go`
- [x] 5.3 Commit (`kit ship`)

## 6. CLI: console, stop, wiring

- [x] 6.1 Red: unit tests for the console lines of design § D10 (each line, with and without color, thousands); `live.serve`: the stopping line with the busy count, workers finish on the work context within the grace time, the pause sleep ends at the stop, the reconnector wired; `start`: the stopped line with and without queued messages
- [x] 6.2 Green: `apps/cli/console_delivery.go`, `live.go`, `start.go`, `start_telegram.go` (Telegram run detached from the stop), `main.go` (first signal stops, second exits 130)
- [x] 6.3 Commit (`kit ship`)

## 7. E2E

- [x] 7.1 Red: `e2e/features/delivery.feature` (`@delivery @e2e`), one `Rule:` per requirement, every E2E scenario of design § Test levels; the world's fakes get LM Studio down/up and an answer delay, Telegram `connection_lost` / `connection_back` / `lost` / `send_hang_ms`, a SIGKILL and a restart over the same database; the changed steps of `message_matching.feature`, `owner_notification.feature` and `catch_up.feature`
- [x] 7.2 Green: fix what the scenarios find
- [x] 7.3 Commit (`kit ship`)

## 8. Docs and version

- [x] 8.1 `README.md`: the retries, the reconnect, the stop and the two settings; `docs/architecture/quality-attributes.html` names the crash tests that now exist
- [x] 8.2 Bump `VERSION` to 0.7.0
- [x] 8.3 Commit (`kit ship`)

Manual checks by the user: none (`manual_run` lists no area); a real Telegram drop over the network stays with the owner's own run.
