## 1. Config file fields: `history_pause`

Files: `tgscout/components/matching/core/domain/settings.go`, `tgscout/components/matching/adapters/secondary/settings/yaml/`, `tgscout.yaml.example`
Needs: none

- [x] 1.1 Red: parser test for "defaults fill what the file leaves out" (adds the 1 s `history_pause` default) and keeps "every form of chat reference is accepted" green
- [x] 1.2 Green: root key `history_pause` (duration, default `1s`), `domain.Settings.HistoryPause` (D7)
- [x] 1.3 Red: parser test for "a negative history pause is an error"; "an invalid config stops the start", "every problem is listed at once", "a retry cap that is not a duration is refused" and "a status interval below 10 seconds is refused" stay green
- [x] 1.4 Green: field error `must be 0s or more` on `history_pause`
- [x] 1.5 Text: `history_pause` in `tgscout.yaml.example` with a one-line comment
- [x] 1.6 Commit the group

## 2. Pace of history reads

Files: `tgscout/infra/telegram/` (`history.go`, `gotd_history.go`, `gotd_client.go`, `fake_client.go`, `new_client.go` config), `tgscout/apps/cli/start_telegram.go`, `tgscout/apps/cli/console_catch_up.go`, `tgscout/apps/cli/live.go`, `e2e/`
Needs: 1

- [x] 2.1 Red: unit test of `telegram.Pacer` with a made-up clock for "history calls keep the pause" (two calls start ≥ the pause apart, a slow call uses up part of it) and "the gap read keeps the pause" (two readers share one pacer)
- [x] 2.2 Green: `telegram.Pacer` (D6); `Config.HistoryPause`; the gotd `ReadHistory` waits on the client's pacer before each `messages.getHistory`; `openTelegram` passes `settings.HistoryPause`
- [x] 2.3 Red: unit test of `telegram.Pacer` for "no pause when it is off"
- [x] 2.4 Green: a pause of 0 returns at once
- [x] 2.5 Red: E2E scenario "history calls keep the pause" in `catch_up.feature` over the fake's `calls_file`
- [x] 2.6 Green: fake Telegram `calls_file` (each `resolve` and `history` call as a JSON line with kind, chat and time) and the fake's `ReadHistory` waits on the same pacer (D7)
- [x] 2.7 Red: console unit test for "the start line names the pause"; "one line per chat", "the connection to Telegram drops" and "a message processed during the drop is not judged twice" stay green
- [x] 2.8 Green: `catchUpStarted(chats, pause)` appends ` · 1s between history reads` when the pause is not 0 (mock)
- [x] 2.9 Text: the scaled catch-up target test sets `history_pause: 17ms` (1 s scaled 60×) and still passes (D6)
- [x] 2.10 Commit the group

## 3. Stored chats and owner

Files: `tgscout/components/matching/core/domain/known_peer.go`, `tgscout/components/matching/core/application/peer_repository.go`, `tgscout/components/matching/adapters/secondary/persistence/sqlite/` (migration `000002_known_peer`, `sqlite_peer_repository.go`), `tgscout/infra/telegram/` (`client.go`, `lookup.go`, `gotd_client.go`, `fake_client.go`, `stub_client_test.go`), `tgscout/apps/cli/` (`connect.go`, `console_telegram.go`, `start.go`, `start_telegram.go`), `e2e/`
Needs: 2

- [x] 3.1 Red: integration test of `SqlitePeerRepository`: `Save`, `FindByRef`, `DeleteByRef` per role and ref; the migration reaches version 2 with the `known_peer` table
- [x] 3.2 Green: migration `000002_known_peer` up and down (D1), `domain.KnownPeer`, the `application.PeerRepository` port and `SqlitePeerRepository` (D2)
- [x] 3.3 Red: `Lookup` unit test for "a chat added to the config is looked up" (two stored chats kept with no `ResolveChat`, the third resolved and saved; counts 2 stored, 1 looked up)
- [x] 3.4 Green: `ChatInfo.Peer`, `Client.KeepChat`, `telegram.KnownPeers` and `Lookup.Known`, `Lookup.Find` returns `LookupCounts` (D3, D4); the gotd client fills `Peer` and keeps stored peers
- [x] 3.5 Red: `Lookup` unit test for "a stored owner whose chat was closed" (stored owner not ready → deleted by ref → `FindOwner` → `OwnerChatClosedError`)
- [x] 3.6 Green: `FindOwner` returns `OwnerInfo`, `Client.CheckOwner`, the owner branch of `Lookup.Find` (D3, D4)
- [x] 3.7 Red: `Lookup` unit tests keep "every chat and the owner are found", "a chat the account has not joined", "Telegram asks to wait during the lookup", "the owner is not found" and "the owner never wrote to tgscout's account" with a store (missing and not-joined chats and a missing owner are never saved); their E2E scenarios stay green
- [x] 3.8 Green: whatever 3.7 shows missing
- [x] 3.9 Red: E2E scenario "a restart looks up no stored chat by name" in `telegram_session.feature` (two starts on one database, the second leaves no `resolve` line in `calls_file` and says every chat came from the store)
- [x] 3.10 Green: the app wires `SqlitePeerRepository` into `Lookup.Known` through a converter; `console.watching(chats, counts)` prints the counts as the mock shows; the fake Telegram implements `KeepChat`, `CheckOwner` and `OwnerInfo`
- [x] 3.11 Commit the group

## 4. A stored chat that stops working

Files: `tgscout/infra/telegram/` (`gotd_errors.go`, `lookup.go`, `fake_client.go`), `tgscout/components/matching/adapters/primary/collect/telegram/history_collector.go`, `tgscout/apps/cli/` (`console_catch_up.go`, `console_delivery.go`, `live.go`), `e2e/`
Needs: 3

- [ ] 4.1 Red: `mapError` unit test: `CHANNEL_INVALID`, `CHANNEL_PRIVATE`, `CHAT_ID_INVALID`, `PEER_ID_INVALID` give `ErrChatGone`
- [ ] 4.2 Green: `ErrChatGone` and its mapping (D5)
- [ ] 4.3 Red: unit test of `HistoryCollector` with `Lookup.Refresh` for "the account left a stored chat" (refresh finds the chat not joined → the read fails, the cursor stays, the repository no longer holds the chat)
- [ ] 4.4 Green: `Lookup.Refresh`, the collector's `Refresher` and `OnRefreshed` report (D5)
- [ ] 4.5 Red: E2E scenario "Telegram no longer knows a stored chat" in `telegram_session.feature` (fake `gone_chats`: the console shows the re-lookup line and the chat's history is judged)
- [ ] 4.6 Green: fake `gone_chats`; console lines `catch-up: <name> · Telegram no longer knows the stored chat · looked it up again` and the `gap read:` twin (mock)
- [ ] 4.7 Commit the group

## 5. Docs and release

Files: `README.md`, `docs/design-progress.md`, `docs/architecture/store.html` (and its page source when one exists), `VERSION`
Needs: 1, 2, 3, 4

- [ ] 5.1 Text: `README.md` names `history_pause` and that found chats and the owner are kept in the database
- [ ] 5.2 Text: a dated line in `docs/design-progress.md` that SQLite also keeps the found chats and the owner, replacing «SQLite keeps only processed IDs and decisions»; the `known_peer` table in `docs/architecture/store.html`
- [ ] 5.3 Bump `VERSION` to 0.9.0
- [ ] 5.4 Commit the group
