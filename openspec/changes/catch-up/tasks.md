## 1. Domain

- [x] 1.1 Red: unit tests for `Settings.CatchUpWindow` (larger messages and days over two requests) and `Settings.CoveringRequests` (rank and days per request; none covers); `CatchUpTally` (expect, settle with matches and a failure, an unexpected key ignored, `Wait` returns once all settled and on ctx end, counts); `Cursor.Hold`/`Release` and `MessageTracker.Hold`/`Release` (a live finish while held does not move; release moves up to the lowest in-flight message)
- [x] 1.2 Green: `core/domain/catch_up.go`, `catch_up_tally.go`, the hold in `cursor.go` and `message_tracker.go`
- [x] 1.3 Commit (`kit ship`)

## 2. Use cases and repository

- [x] 2.1 Red: unit tests for `CollectMessage.CollectHistory` (queued with the given requests and expected in the tally; processed, in flight, no text skipped), the `CatchUp` use case (`Open` holds and gives the window with and without a stored cursor, `Close` releases and stores a moved cursor, `Wait` gives the counts), `MatchMessage` settling the tally (a match counted, a failure counted, an edit not settled); integration test for `SqliteMessageRepository.Cursor` (found with its time, not found)
- [x] 2.2 Green: `core/application/collect_message.go` (shared `admit`), `catch_up.go`, `match_message.go`, `message_repository.go` (`Cursor`), the SQLite repository, `matching_config.go` (shared tally, the `CatchUp` use case)
- [x] 2.3 Commit (`kit ship`)

## 3. Telegram client

- [x] 3.1 Red: unit tests for `cutWindow` (after, limit, since, gap with each limit, oldest first); the fake's `ReadHistory` (`history`, `history_flood_wait_seconds` once, `history_delay_ms`); the gotd history paging over a stubbed invoker where gotd allows it; the stub client gets `ReadHistory`
- [x] 3.2 Green: `infra/telegram` `HistoryWindow`, `History`, `ReadHistory` in the port, `history.go` (`cutWindow`), the gotd client (kept input peers, `messages.getHistory` paging), the fake client
- [x] 3.3 Commit (`kit ship`)

## 4. History collector

- [x] 4.1 Red: `components/matching/tests/features/slice/catch_up.feature` (`@catch_up @slice`): the slice scenarios of design § Test levels through the history collector over a stub client, the real use cases and the memory repository; integration test of the collector over a stub client (FLOOD_WAIT retried, a failed read goes on with the next chat, the per-chat report)
- [x] 4.2 Green: `adapters/primary/collect/telegram/history_collector.go` (shared update mapping with the live collector)
- [x] 4.3 Commit (`kit ship`)

## 5. CLI

- [x] 5.1 Red: unit tests for the console lines of design § D5 (start line, chat line from the cursor, first start, gap, the summary with and without errors, a chat with no title) and for `live.serve` running catch-up beside listening
- [x] 5.2 Green: `apps/cli/console_catch_up.go`, `live.go`
- [x] 5.3 Commit (`kit ship`)

## 6. E2E

- [x] 6.1 Red: `e2e/features/catch_up.feature` (`@catch_up @e2e`), one `Rule:` per requirement, every E2E scenario of design § Test levels; the world's fake Telegram gets `history`, `history_flood_wait_seconds` and `history_delay_ms`
- [x] 6.2 Green: fix what the scenarios find
- [x] 6.3 Commit (`kit ship`)

## 7. Docs and version

- [x] 7.1 `docs/architecture/context.html` and `quality-attributes.html` (design § D6); `README.md`: catch-up at start, its console lines
- [x] 7.2 Bump `VERSION` to 0.6.0
- [x] 7.3 Commit (`kit ship`)

Manual checks by the user: none for this run (`manual_run` lists no area); the owner's run of the acceptance bar after a pause stays for the owner.
