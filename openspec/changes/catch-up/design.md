## Context

live-matching gave tgscout the message path for new messages: the collector maps a Telegram update to a `domain.Message`, `CollectMessage` drops what needs no judgment and queues the rest, the workers run `MatchMessage` (LLM, decision, private message, processed mark, cursor). The `MessageTracker` holds the in-flight messages and the in-memory cursor of each chat. The settings already resolve a catch-up depth per request and chat (`domain.RequestChat.CatchUp`, 500 messages and 3 days by default). Nothing reads history yet, and nothing reads the stored cursor.

## Goals / Non-Goals

**Goals:** read the missed history of every watched chat at start, beside live listening, through the same path; per-request depth; no double judging; a cursor that never passes unread history; console lines and a summary.

**Non-Goals:** deep history search (version 2), the 15-minute target on 100 chats (`quality-targets`), reconnect gaps while running (`delivery-resilience`).

## Decisions

### D1. History read in the infra client

`telegram.Client` gets one read call:

```go
// HistoryWindow is how far back a history read goes: messages after After,
// at most Limit of them, none older than Since.
type HistoryWindow struct { After, Limit int; Since time.Time }
// History is a read: the messages oldest first; Gap is true when the window
// stopped before it reached the message after After.
type History struct { Messages []Update; Gap bool }
ReadHistory(ctx context.Context, chatID int64, window HistoryWindow) (History, error)
```

One shared helper `cutWindow(newestFirst []Update, window) History` applies the window for both clients, so the gotd client and the fake one agree. The gotd client pages `messages.getHistory` (100 per page, `min_id = After`, `offset_id` = lowest ID of the last page) newest first until a page is empty, `Limit + 1` messages are in hand or a message is older than `Since`; the extra message only tells the gap. It needs the input peer of the chat: `ResolveChat` now keeps it by bare ID (`InputPeerChannel` with its access hash from the resolved username, or the dialog's peer for a numeric ID), as `FindOwner` keeps the owner's. A FLOOD_WAIT comes back as `*FloodWait`; the caller retries with the existing `telegram.Retry`.

The fake client gets `history: [{chat, id, text, service, age_minutes}]`, `history_flood_wait_seconds` (once) and `history_delay_ms` (a pause before the read returns, so a live update can arrive during the read).

### D2. The catch-up use case and the depth rules

Domain:

- `Settings.CatchUpWindow(ref) CatchUp` — the larger messages and the larger days over the requests of the chat.
- `Settings.CoveringRequests(ref, rank, sentAt, start) []Request` — the requests of the chat whose own depth covers a message: `rank <= Messages` (rank 1 is the newest message read) and `sentAt >= start − Days`.
- `CatchUpTally` — the history messages handed to the judge and not yet settled, and the counts (read, skipped, judged, matches, errors); `Expect(key)`, `Settle(key, matches, failed)`, `Wait(ctx)`.
- `Cursor` gets `Hold()` and `Release() (int, bool)`: while held, `Finish` records the finished message but never moves; `Release` moves the cursor as far as the finished and in-flight messages allow. `MessageTracker.Hold(chatID)` / `Release(chatID)` pass through.

Application:

- `CollectMessage.CollectHistory(ctx, message, requests) (bool, error)` — the same path as `Collect` (text check, `Begin`, `IsProcessed`, `Put`) over the given covering requests; it calls `tally.Expect` before `Put`; false means skipped. `Collect` and `CollectHistory` share one private `admit`.
- `CatchUp` use case: `Open(ctx, chatID, ref) (domain.CatchUpChat, error)` holds the chat's cursor, reads the stored cursor (new `MessageRepository.Cursor(ctx, chatID) (id int, at time.Time, found bool, err error)`) and returns the window; `Close(ctx, chatID) error` releases the hold and stores the cursor when it moved; `Wait(ctx) (domain.CatchUpCounts, error)` waits for the tally.
- `MatchMessage` settles the tally at the end of every judgment of a new (not edited) message: the number of matches sent and whether it failed. A message not expected is not counted.

The start time of the depth is taken once, when catch-up begins.

### D3. The history collector (primary adapter)

`collect/telegram.HistoryCollector.CatchUp(ctx, client, chats []Chat)` takes the chats in config order (each `Chat` gets its bare ID), and for each: `Open`, `ReadHistory` through `Retry`, maps every update with the live mapping (title, link; one shared `message` func), ranks them from the newest, `CollectHistory` with the covering requests (a message nobody covers counts as skipped), `Close`, then the chat's console line. After the last chat it waits for the tally and reports the summary. A history read that fails reports the chat as failed and goes on with the next chat; its cursor is released unmoved.

### D4. Start flow

`live.serve` starts the workers, prints the running line, then starts catch-up in its own goroutine beside `collector.Listen`; the watched chats keep the config order (`watchedChats`). The catch-up goroutine ends with ctx; serve waits for it before it returns. A stop before the summary prints no summary.

### D5. Console

The console follows the style the owner-notification run settled (no clock or level words, the existing marks):

```
→ catch-up started · 3 chats · live listening is on
✓ catch-up: Riverside flea market · 214 messages from the cursor
✓ catch-up: Kids' things swap · 500 messages · first start, last 3 days
⚠ catch-up: Bikes for sale · 500 messages · off for 12 days, read only the last 3 days
✓ catch-up done in 4m35s
  ✓ chats      3  (1 with a gap)
  ✓ messages   1,042 read · 18 skipped · 1,024 judged
  ● matches    2  (sent to you in Telegram)
  ✓ errors     0
```

Numbers over 999 get thousands commas, as in the mock.

The errors line gets the fail mark and «(not marked processed; judged again at next start)» when not 0. "Off for N days" counts from the stored cursor's time. The depth text is «last N days» with the chat's window days. A FLOOD_WAIT prints the existing wait line «⚠ Telegram asks to wait 1s (FLOOD_WAIT), then goes on».

Decided by the prepared run, beside the approved mock: the mock's clock and INFO/WARN words are left out (as owner-notification did), and «skipped» drops the mock's «(already processed)», since it also counts service messages, messages with no text and messages no request covers; the gap line keeps the count of messages read («500 messages ·») before the mock's «off for 12 days, read only the last 3 days», as the spec's chat line asks for the count.

The cursor hold (D2) modifies the main requirement message-matching «Processed mark and cursor» (delta in `specs/message-matching/spec.md`): a live message of a chat being caught up no longer moves the cursor at once. This follows the prepared decision «It waits until that chat's catch-up is done, then moves».

### D6. Docs

`docs/architecture/context.html`: the line «history matches come as one summary message» becomes «each history match comes as its own private message; the summary goes to the console». `docs/architecture/quality-attributes.html`: the catch-up time is measured «start time → summary printed time».

## Test levels

| Scenario | Level | Where |
|---|---|---|
| a match in the history reaches the owner | E2E | `e2e/features/catch_up.feature` (`@catch_up @e2e`): fake history, the sent file |
| first start reads the default depth | E2E | feature: history over 4 days; the LLM calls hold only the last 3 days, oldest first |
| a restart reads only what it missed | E2E | feature: a first start over 10 messages sets the cursor; the second start over 30 judges 20 |
| the message limit comes first | slice | `components/matching/tests/features/slice/catch_up.feature` (`@catch_up @slice`): the history collector over a stub client with 700 messages, real use cases, memory repository |
| a cursor older than the depth | slice | slice feature: a stored cursor of 12 days ago |
| a request overrides the depth | slice | slice feature |
| a chat overrides the depth | slice | slice feature |
| two requests with different depths on one chat | slice | slice feature |
| a processed message is not judged again | slice | slice feature: the memory repository holds a processed mark after the cursor |
| live and catch-up at once | E2E | feature: `history_delay_ms` and a live update with the ID of a history message; the LLM is asked once |
| a new message does not move the cursor past unread history | unit | `Cursor` / `MessageTracker` tests: the order of a live finish and a history read cannot be forced from outside |
| message-matching: the cursor never passes an unprocessed message | unit | unchanged `MessageTracker` and `MatchMessage` tests; the hold adds the `Cursor` tests above |
| Telegram asks to wait during catch-up | E2E | feature: `history_flood_wait_seconds: 1` |
| one line per chat | E2E | feature |
| the summary | E2E | feature: three chats |
| judging errors are counted | E2E | feature: the fake LM Studio answers 503 |

Unit tests: `cutWindow`, the domain depth functions, `CatchUpTally`, `Cursor` hold, `CollectHistory`, the `CatchUp` use case and `MatchMessage` settling with fake ports, the console lines. Integration (`-tags=integration`): `SqliteMessageRepository.Cursor` on a temp file; the history collector over a stub `Client` through a real `ReadHistory` call (FLOOD_WAIT retried). The gotd `ReadHistory` paging gets a unit test over a stubbed `messages.getHistory` invoker where gotd allows it; the real network path stays with the owner's manual run, as the override `telegram-sign-in` named.

## Risks / Trade-offs

- A history read holds up to the depth's messages in memory (500 by default, a few thousand with an override) — small.
- Each chat's history is read with one `getHistory` page loop; 100 chats × 5 pages stays well under Telegram's limits, and FLOOD_WAIT is waited out.
- The «off for N days» number is the time the cursor last moved, not the time tgscout stopped — close enough for the console.

## Interface mock

[Catch-up console mock](mock/index.html) — the start line, one line per chat, the summary; approved in the preparation (D5 lists what the run changed).

## Release

MINOR, 0.5.0 → 0.6.0, bumped in the last task group.
