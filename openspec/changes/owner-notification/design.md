## Context

`live-matching` (v0.4.0) judges each new message of a watched chat and hands every «yes» to the `Notifier` port; the only adapter prints a match line on the console. `MatchMessage` already calls `Notify`, then `MarkSent` (`sent_at`), and marks the message processed and moves the cursor only after the last request. `telegram-sign-in` gives one MTProto client (`infra/telegram.Client`) for the whole run, the lookup of the owner (`FindOwner`) and a FLOOD_WAIT retry inside the lookup. This change makes Telegram the delivery: each match goes to the owner as a private message from tgscout's account.

## Goals / Non-Goals

**Goals:** the Telegram notifier, the text of the private message, send-then-mark with the confirmed send, FLOOD_WAIT on a send, the start check that the owner's private chat is open, writing only to the owner.

**Non-Goals:** owner feedback and commands; the catch-up summary (`catch-up`); retries after a long outage and crash tests (`delivery-resilience`).

## Decisions

### 1. Text of the private message (domain)

`domain.NotificationText(match Match) string` builds the four lines: `Match: <id> — <find>`, `Chat: <title>`, the link, the text. `Message` gets `ChatTitle`; an empty title falls back to `Chat.String()` (the config text). The text is cut by runes: more than 1,000 gives the first 1,000 and `…`. Line breaks inside the text stay (Telegram shows them). Pure function, unit-tested.

### 2. Chat titles

`telegram.ChatInfo` gets `Title`. The gotd lookup takes it from the resolved `tg.Channel`/`tg.Chat`; a numeric reference takes it from the dialog's entities (`dialogs.Elem.Entities.Channel/Chat`). The collector's `Chat` gets `Title` and puts it into `Message.ChatTitle`; `apps/cli/live.go` passes it from the found chats. The fake client takes titles from a scenario map `titles`.

### 3. Client port: owner state and send

- `FindOwner(ctx, owner) (OwnerState, error)` replaces the boolean: `OwnerMissing`, `OwnerNoChat`, `OwnerReady`. The gotd client keeps the owner's `InputPeerUser` (from the resolved username or the contacts list) and asks `messages.getPeerDialogs` for it: no dialog, or a dialog with no top message, is `OwnerNoChat`. `Lookup.Find` turns `OwnerNoChat` into `OwnerChatClosedError`; the start prints «cannot write to the owner» and the hint «write «hi» to tgscout's account once and add it to contacts (README → Setup → First contact)», then exits 1.
- `SendToOwner(ctx, owner, text) error` sends one plain message (`messages.sendMessage`, a random ID from `crypto/rand`, link previews off) to the kept owner peer; with no kept peer it finds the owner first. Errors go through `mapError`: a FLOOD_WAIT becomes `*FloodWait`, any other server error keeps no server text. This is the only call of the port that writes, and it takes no peer from outside: nothing else can be written to.
- The FLOOD_WAIT retry of the lookup becomes the exported `telegram.Retry(ctx, waiting, sleep, call)`; the lookup and the notifier share it. No cap on the wait (preparation).

### 4. `TelegramNotifierClient` (secondary adapter)

`adapters/secondary/notify/telegram`: `NewTelegramNotifierClient(client telegram.Client, owner string, printer MatchPrinter, waiting func(time.Duration), sleep func(ctx, d) error)`. `Notify` builds the text, sends it through `telegram.Retry`, and only after the confirmed send hands the match and its text to the printer (the console). A console write that fails after a confirmed send is not a send failure: the adapter drops that error, so the owner is never sent the same match twice for a lost console line. The console adapter `adapters/secondary/notify/console` and its test are removed: the app wires the Telegram notifier instead.

### 5. Send-then-mark

The order of `MatchMessage` stays: decision saved → `Notify` (returns only after Telegram confirmed) → `MarkSent` → after the last request `MarkProcessed` → cursor. A stop during the send or the FLOOD_WAIT returns the context error from `Notify`, so the message stays unprocessed and in the tracker's failed set; the next start judges it again (at least once). `StageNotify` prints as `send`: the error line reads «✗ send failed on <chat>/<id>: <cause>» with the hint «the message stays unprocessed».

### 6. Console

Following the approved mock in the console's own style (marks, no log levels or clock prefix, as every other line of tgscout):

```
✓ match sent  10:02  bike-16
  Match: bike-16 — kids' bike, 16 inch, under 100 €
  Chat: Riverside flea market
  https://t.me/c/1234567890/4521
  Selling a kids' bike, 16", blue, good state. 80 €. Pick up near the station.
⚠ Telegram asks to wait 5s (FLOOD_WAIT), then sends again
✗ send failed on @made_up_market/4529: cannot reach Telegram
  the message stays unprocessed
✗ cannot write to the owner
  write «hi» to tgscout's account once and add it to contacts (README → Setup → First contact)
```

The mock's «INFO»/«ERROR» level words and clock prefix are not used: no other line of tgscout has them (decided by the prepared run). The mock's «(text cut at 1,000 characters)» note is the mock's own caption; the cut text ends with «…» as the preparation says (decided by the prepared run).

### 7. Writing only to the owner

The gotd update mapping already drops every message of a private chat with a user (`mapMessage` gives false for `PeerUser`), so a private message, the owner's own included, is never judged nor answered; the port has no call that writes to anyone but the owner. The fake client mirrors the drop for scenario updates marked `private`.

## Test levels

| Scenario | Level | Where |
|---|---|---|
| a match reaches the owner | E2E | `e2e/features/owner_notification.feature` (`@owner_notification @e2e`): the fake Telegram records every send in the scenario's `sent_file` |
| two requests match one message | E2E | same feature |
| a long message text is cut | unit | `domain.NotificationText` test (1,500 characters; the link and the request stay whole); the console and Telegram get the same text, so no E2E run is needed for the cut |
| the message is marked processed only after the send | slice | `components/matching/tests/features/slice/owner_notification.feature` (`@owner_notification @slice`): real `MatchMessage` and `TelegramNotifierClient`, a stub client whose send blocks until the stop, the memory repository: the stop inside a send cannot be timed from outside the binary |
| Telegram asks to wait | slice | same slice feature: the stub client answers FLOOD_WAIT 5s once; the injected sleep records 5s without waiting it |
| a send fails | E2E | owner feature: the fake fails every send (`send_fails`) |
| tgscout writes only to the owner | E2E | owner feature: the fake plays a private message from a made-up stranger; nothing is sent; plus the unit test of `mapMessage` for `PeerUser` |
| message-matching: a matching message is shown on the console | E2E | `e2e/features/message_matching.feature`, steps updated to the new match line |
| message-matching: a message that does not match | E2E | `e2e/features/message_matching.feature`, unchanged |
| message-matching: the cursor never passes an unprocessed message | unit | `MessageTracker` and `MatchMessage` tests, unchanged |
| message-matching: an edited message is judged again only where it was «no» | E2E | `e2e/features/message_matching.feature`, steps updated to the new match line |
| message-matching: a match saved but not shown is shown again | unit | `MatchMessage` test, unchanged |
| telegram-session: the owner is not found | E2E | `e2e/features/telegram_session.feature`, unchanged |
| telegram-session: the owner never wrote to tgscout's account | E2E | `e2e/features/telegram_session.feature`, the fake's `owner_chat_closed` |

Integration test (`-tags=integration`): `TelegramNotifierClient` over a stub `telegram.Client` and a buffer printer — the text sent, the printer called after the send, a FLOOD_WAIT retried, a send error returned with no print, a printer error dropped after the send. Unit tests: `NotificationText`, `Lookup` with each `OwnerState`, `Retry`, the fake client's sends, the console lines.

The gotd network path of `SendToOwner` and the dialog check get no automated test against real Telegram, as for the rest of `infra/telegram` (`telegram-sign-in` override); the owner's manual run of the acceptance bar covers it.

## Risks / Trade-offs

- [Telegram may refuse a stranger's message (PEER_FLOOD) even after «hi»] → the start check and the README; a refused send is an error line and the message stays unprocessed.
- [A stop between the confirmed send and `MarkSent` sends the match again at the next start] → allowed by the proposal (at least once).
- [`messages.getPeerDialogs` behavior for a user with no history] → treated as «no chat» both when no dialog and when its top message is 0.
- [A failed send blocks the chat's cursor for the rest of the run] → as for an LLM failure; `delivery-resilience` adds the retry.

## Interface mock

[Owner notification mock](mock/index.html) — the private message in Telegram light and dark, and the console lines; approved in the preparation.

## Release

MINOR, 0.4.0 → 0.5.0, bumped in the last task group.
