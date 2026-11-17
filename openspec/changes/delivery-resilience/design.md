## Context

The earlier slices put the delivery order in place: judge, save the decision, send, mark sent, then mark processed and move the cursor. A failure anywhere leaves the message unprocessed, so the next start judges it again. What is missing is the behavior while tgscout runs: today one failed LLM call or send prints an error and the message stays in flight until the next start; a stop cancels every call at once; a dropped Telegram connection is left to gotd, whose own gap recovery we do not rely on; nothing tests a kill.

## Goals / Non-Goals

**Goals:** retries with growing pauses for LM Studio and for the send; one status line when LM Studio goes and one when it is back; a line when the queue is full; a gap read from the cursors after a reconnect; a graceful stop with a grace time; two crash tests.

**Non-Goals:** a queue on disk; running as a system service; alerts outside the console; showing that a repeated match after a crash is a repeat.

## Decisions

### D1. Backoff in the domain

`domain.Backoff{Max time.Duration}` gives the pause before try n+1: `Pause(n) = min(1s << (n-1), Max)`, so 1 s, 2 s, 4 s … up to `retry_max_pause`. A pure value, unit tested.

### D2. LLM retries in `MatchMessage`

`MatchMessage` asks the LLM in a loop. The `Llm` port's error decides the loop:

- an error that wraps `application.ErrLlmRejected` (the answer is about this message: an HTTP 4xx, an answer that cannot be decoded, no choice) counts a try; after 10 tries the decision is saved as «no» and `GaveUp` is reported, then the next request of the message goes on;
- any other error (no connection, the timeout, an HTTP 5xx) is "no answer": retried without end. A long outage never turns a match into «no» (proposal § Decisions);
- a context error ends the judgment: the message stays unprocessed.

The `OpenAiLlmClient` wraps `ErrLlmRejected` for 4xx and unreadable answers; the reach errors and 5xx stay as they are.

A small `llmHealth` inside the use case (one per `MatchMessage`, shared by all workers, a mutex and a bool) reports `LlmDown` on the first "no answer" while up and `LlmBack` on the first success while down, so the console shows one line each way however many workers retry.

`NewMatchMessage` takes a `Delivery` value: the `Backoff`, a `Sleep func(ctx, d) error` and a `DeliveryReporter` port (`LlmDown(max)`, `LlmBack()`, `GaveUp(item, request, tries, err)`, `SendFailed(match, err, pause)`). The app passes the console's methods through `DeliveryFuncs`, as `ReportFuncs` does today.

### D3. Send retries in `MatchMessage`

The notifier keeps waiting out FLOOD_WAIT (owner-notification). Any other error it returns is retried by `MatchMessage` with the same backoff, without end, reporting `SendFailed` before each pause. `MatchMessage` owns both loops so one place holds the retry rule; the adapter stays a thin send.

### D4. Pauses end at the stop

The pause between tries uses `Delivery.Sleep`. The app builds it in `live.serve` so it also ends when the stop arrives: a worker in a pause gives its message up at once (unprocessed), and only a call already on its way gets the grace time (D7).

### D5. Full queue line

`ChannelMessageQueue` gets an `OnFull func(size int)` and `Len()`. `Put` tries a send without blocking first; when the queue is full it calls `OnFull` once, then blocks as today. The flag is cleared when `Receive` leaves the queue empty, so the next full episode prints again. The console prints `⚠ queue is full (1,000) · reading new messages paused`.

### D6. Reconnect and gap read

- `telegram.Client` gains `WatchConnection(f func(ConnectionEvent))`, with `ConnectionLost` and `ConnectionBack`. The gotd client sets `telegram.Options.OnConnectionState` at build time and calls `f` with Lost on `ConnectionStateDisconnected` and with Back on the next `ConnectionStateReady` after a Lost (the first Ready of the run is no event). The callback must not block: it only hands the event to a buffered channel.
- The fake client plays three new update fields: `connection_lost`, `connection_back` and `lost` (an update posted while the connection is away: not delivered live, added to the history the fake serves from then on).
- A new primary adapter piece, `collect/telegram/reconnect.go` `Reconnector`, reads the events. On Lost it reports and calls `CatchUp.Pause(chatIDs)`, which holds every watched chat's cursor. On Back it reports and runs `HistoryCollector.GapRead(ctx, client, chats)`: the same per-chat read as catch-up (Open holds, read after the stored cursor, Collect, Close releases), then one report with the number of messages handed to the judge. A Back while a gap read runs makes one more read after it ends.
- `CatchUp.Collect` returns whether the message was handed to the judge, so the gap read can count it.
- Decided by the prepared run: gotd's updates manager (its own update-gap recovery) stays in place, beside our gap read. It replaces the letter of the preparation decision «not the MTProto library's gap recovery»; the meaning holds, since tgscout never relies on it: the cursor read is the recovery. gotd's manager could be dropped by passing the dispatcher straight as the update handler, but the plain dispatcher does not handle the short update forms (`updateShortChatMessage`) that small groups use, so dropping it would lose live messages on a path only the owner's manual run can check. A message the manager delivers as well is dropped by the tracker and the processed check, so it is judged once.

### D7. Graceful stop

- `main` reads the signals itself: the first SIGINT/SIGTERM cancels the run's context; a second one calls `os.Exit(130)` at once.
- `watch` runs the Telegram client on a context detached from the stop (`context.WithoutCancel`), ended when `serve` returns, so a send in work can still reach Telegram during the grace time.
- In `serve`, the listener, the catch-up, the gap reads and the workers' `Receive` use the stop context. The workers' `Match` uses a work context that ends `stop_grace` after the stop. `RunWorkers` takes both contexts and counts the busy workers.
- At the stop `serve` prints `→ stopping · finishing 2 messages in work · up to 10s · Ctrl+C again to stop now`; after the database is closed `start` prints `✓ stopped · 18 messages in the queue left unprocessed, catch-up reads them next start`, or `✓ stopped` when the queue is empty. A stop before `serve` runs keeps today's `→ stopping…`.

### D8. A failed history read keeps the cursor

`catchUpChat` closes the chat (releases its cursor) only after a read that worked. After a failed read the cursor stays held for the rest of the run, so live messages that finish later never move it past the unread history; the next start reads it again. This fixes a loss the catch-up slice left (related work, catch-up spec).

### D9. Settings

`domain.Settings` gains `RetryMaxPause` (default 60 s) and `StopGrace` (default 10 s); the YAML parser reads `retry_max_pause` and `stop_grace` with the same duration rule as `lm_studio.timeout` and refuses a value that is not a positive duration. `tgscout.yaml.example` and the README name both.

### D10. Console lines

Decided by the prepared run: the approved mock shows the lines with a time and a level word (`10:14:02 WARN`); tgscout's console has neither — it prints a mark (`✓ ⚠ ✗ →`, or `[ok] [warn] [fail] ->` without color). The lines keep the mock's words in the console's own form:

| Event | Line | Stream |
|---|---|---|
| LM Studio goes | `⚠ LM Studio is not answering · retry in 1s, 2s, 4s … up to 60s · messages wait in the queue` | stderr |
| LM Studio is back | `✓ LM Studio is back · 1,000 messages waiting` | stdout |
| queue full | `⚠ queue is full (1,000) · reading new messages paused` | stderr |
| gave up | `⚠ judge gave up after 10 tries: bike-16 on @made_up_market/4610 · saved as no · <reason>` | stderr |
| send failed | `⚠ send failed: bike-16 on @made_up_market/4529 · <reason> · retry in 2s · the message stays unprocessed` | stderr |
| connection lost | `⚠ Telegram connection lost · reconnecting` | stderr |
| connection back | `✓ Telegram connection back · reading 3 chats from their cursors` | stdout |
| gap read done | `✓ gap read done · 42 messages to judge` | stdout |
| stopping | `→ stopping · finishing 2 messages in work · up to 10s · Ctrl+C again to stop now` | stdout |
| stopped | `✓ stopped · 18 messages in the queue left unprocessed, catch-up reads them next start` | stdout |

The match line stays owner-notification's `✓ match sent  HH:MM  <id>`. The mock's «sent before the crash, sent again» note is not shown: the use case cannot tell a repeat from a first send without a new mark, and the proposal allows the repeat (Non-Goals).

## Test levels

| Scenario | Level | Where |
|---|---|---|
| LM Studio goes away and comes back | E2E | `e2e/features/delivery.feature` (`@delivery @e2e`): the fake LM Studio answers 503, then «yes»; the owner gets the match |
| the pauses grow up to the cap | unit | `MatchMessage` test with a recording sleep |
| a message the model never manages | unit | `MatchMessage` test: a rejecting fake LLM, 10 tries, «no» saved, processed, `GaveUp` reported |
| no answer never gives up | unit | `MatchMessage` test: 12 no-answers, then «yes» |
| one line when it goes, one when it is back | unit | `MatchMessage` test with two messages and a recording reporter; console line test |
| a send fails, then works | slice | `tests/features/slice/owner_notification.feature`: the stub client fails twice, then sends |
| the queue is full | E2E | feature: `queue_size: 1`, one worker, LM Studio down, three messages; the full line shows, then all three are judged after LM Studio is back |
| the cursor waits while the connection is away | unit | `CatchUp.Pause` and `MessageTracker` tests |
| the connection to Telegram drops | E2E | feature: the fake plays `connection_lost`, a `lost` update, `connection_back`; the lost message is judged |
| a message processed during the drop is not judged twice | slice | `tests/features/slice/catch_up.feature`: the gap read over a stub client whose history holds a processed message |
| the owner stops tgscout during work | E2E | feature: the fake LM Studio answers after 1 s; Ctrl+C while it judges; exit 0, the stopping line |
| a judgment longer than the grace time | E2E | feature: `stop_grace: 1s`, the fake LM Studio answers after 5 s; exit 0 in under 3 s, not processed |
| a second Ctrl+C stops at once | E2E | feature: the fake LM Studio answers after 5 s; two SIGINTs; exit 130 |
| a crash between send and mark | E2E | feature: the fake Telegram holds `send_hang_ms` after it wrote the sent line; SIGKILL; the restart's history holds the message; two sends of it |
| a crash with messages in the queue | E2E | feature: one worker, the fake LM Studio hangs, three messages; SIGKILL; the restart's history holds them; all three asked again |
| a clean restart repeats nothing | E2E | feature: a match, Ctrl+C, a restart whose history holds the message; one send, the LLM not asked again |
| message-matching: LM Studio is down | E2E | `message_matching.feature`: the step now looks for the not-answering line |
| owner-notification: a send fails | E2E | `owner_notification.feature`: every send fails; a warning line per try, unprocessed |
| the cursor never passes an unprocessed message | unit | unchanged `MessageTracker` / `MatchMessage` tests; the hold during a drop adds the `CatchUp.Pause` tests |
| an invalid config stops the start | E2E | unchanged `settings.feature` outline |
| every problem is listed at once | E2E | unchanged `settings.feature` |
| a new message does not move the cursor past unread history | unit | unchanged `Cursor` / `MessageTracker` tests |
| tgscout starts and stops cleanly | E2E | unchanged `cli_lifecycle.feature`; the stopped line still holds «stopped» |
| tgscout stops on a terminate signal | E2E | unchanged `cli_lifecycle.feature` |
| defaults fill what the file leaves out | unit | settings parser test: 60 s and 10 s added |
| every form of chat reference is accepted | unit | unchanged settings parser test |
| a retry cap that is not a duration is refused | unit | settings parser test |
| a failed history read keeps the cursor | unit | `HistoryCollector` test over a stub client whose read fails |
| catch-up: judging errors are counted | unit | `MatchMessage` settling test with a failing repository (a failed database write) |
| catch-up: the summary waits for LM Studio | E2E | `catch_up.feature`: the old «judging errors» scenario, now no summary while the fake answers 503 |

Unit tests also cover `Backoff`, the queue's `OnFull` and `Len`, the console lines, the gotd connection-state mapping (a function over `ConnectionState` values) and the fake's lost updates. The real reconnect path over the network stays with the owner, as `telegram-sign-in` decided for every gotd network path.

## Risks / Trade-offs

- A drop while startup catch-up reads a chat: that chat's Close releases the hold before the connection is back. Only history and queued messages can finish while the connection is away, and they are older than the gap, so the cursor cannot pass a gap message.
- Workers in a long LM Studio outage hold their messages and retry; the collector blocks on the full queue inside gotd's update handler. gotd buffers and its own gap recovery may refetch; the tracker drops the copies.
- The E2E suite gets waits of 1–5 s per scenario (retry pauses, grace time); the full check shows whether the time budget moves.

## Interface mock

[Delivery resilience console mock](mock/index.html) — the lines for LM Studio, a failed send, a gave-up message, a dropped connection, a stop and a start after a crash; approved in the preparation (D10 lists what the run changed).

## Release

MINOR, 0.6.0 → 0.7.0, bumped in the last task group.
