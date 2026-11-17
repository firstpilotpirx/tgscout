## Context

See proposal.md § Why. The matching flow today: the live listener and the history reads hand messages to `CollectMessage`, which queues them; the workers run `MatchMessage`, which asks the LLM per request, saves each decision and sends each match through the notifier, which prints the match line after Telegram confirms. Nothing counts or times this flow. The console is one `console` type in `apps/cli` with status marks (`✓ ✗ ⚠ ● →`, plain `[ok]` … without color) and indented detail lines.

## Goals / Non-Goals

**Goals:** count and time the flow in one place, print it every `status_interval` and at stop, mark the targets, warn when the model falls behind and when the LLM is not local; prove the speed, catch-up and size targets with tests that run before merge.

**Non-Goals:** metrics export, history of past status lines, alerts outside the console, tuning the real model.

## Decisions

### D1. `domain.RunStats` holds the numbers

A domain type with a mutex records, per interval and since start: messages in, messages judged, matches sent, the latencies of live messages and the LLM call times. `Cut(queue, now)` gives the `Status` of the last interval and starts a new one; `Total(queue, now)` gives the `Status` since start. `Status` holds the counts, the queue length, how long it covers and the two p95 values (`ok` false when there is no sample). Methods on a nil `*RunStats` do nothing, so tests and callers that do not count need nothing.

p95 is the nearest-rank percentile (`domain.P95`): sort, take the value at rank ⌈0.95·n⌉. Samples are kept as durations in a slice per interval and one for the whole run; at 5,000 messages a day the run slice grows by about 40 KB a day, fine for a console run (the alternative, a histogram, is exact enough but harder to read and test).

Alternative: counters in the app's console. Rejected: the use cases know when a message is judged and how long a call took; the app does not.

### D2. Where the counts come from

- `domain.Message` gets `Live bool`: `CollectMessage.Collect` (the live listener's path) sets it for a message that is not an edit; history reads never set it.
- `CollectMessage.Collect` counts `In` for a live message it queues (dropped duplicates and edits are not counted).
- `MatchMessage.ask` times each `Llm.Judge` call that ends (an answer or an error) and records it.
- `MatchMessage.Match` counts `Judged` when a judgment ends without error, with the latency `now − SentAt` when the message is live; it counts `Matched` after each confirmed send.

`matching.Ports` gets `Stats *domain.RunStats`; `NewMatchingConfig` passes it to both use cases.

### D3. Latency on the match line

The notifier prints the match after Telegram confirms (owner-notification). `console.PrintMatch` adds ` · latency 3.4s` when `match.Message.Live`, with the latency `time.Since(SentAt)` rounded to 0.1 s. Telegram dates have whole seconds, so a real latency may read up to 1 s high: the target check errs on the safe side.

### D4. The status loop in the app

`live.serve` starts a status goroutine with a ticker of `status_interval`: each tick `stats.Cut(queue.Len(), now)` → `console.status(s)`, then a `queueTrend` (the last queue lengths) decides the behind warning: three lines in a row each longer than the one before; after a warning the trend starts over, so the next warning needs three more growing lines. The loop is a function over a tick channel, so a unit test drives it with made-up ticks and stats.

At stop, `start` prints `console.statusTotal(stats.Total(...))` after `serve` returned (the work is finished) and before the stopped line. A stop before the workers ran prints no status.

Lines (plain marks shown; the mock's content in the existing style, proposal § Decided by the prepared run):

```
[run] status · last 5m
  in 312 · judged 1,904 · matches 2 · queue 4
  p95 latency 3.8s [ok] ≤ 10s · p95 LLM call 1.1s
[warn] the model is behind: the queue grew 3 status lines in a row (97 → 214 → 388)
  use a faster model or more workers
[ok] status since start (8h33m)
  in 4,980 · judged 29,410 · matches 9 · queue 0
  p95 latency 4.6s [ok] ≤ 10s · p95 LLM call 1.2s
```

Over the target: `[fail] > 10s`; no sample: `p95 latency —`.

### D5. Catch-up target mark

`catchUpDone` writes `catch-up done in 13s · [ok] ≤ 15m`, or `· [warn] > 15m — the model answers too slowly for this many messages`. The target is a constant of the app (`catchUpTarget = 15 * time.Minute`), as is the latency target (`latencyTarget = 10 * time.Second`).

### D6. Non-local LLM

`domain.LmStudio.Local()` parses the URL host: `localhost` (any case), `127.0.0.1`, `::1` are local. `loadConfig` prints a warning block after the summary when it is not local and goes on.

### D7. `status_interval` setting

A root duration key in the YAML parser, default `5m`; below `10s` is a field error `must be at least 10s`. `domain.Settings.StatusInterval`.

### D8. Test levels

| Spec scenario | Level |
|---|---|
| latency on a match line | E2E (`quality_targets.feature`) |
| no latency on a catch-up match | E2E |
| the status line | E2E, status interval 10 s |
| an interval without live messages | unit (console + stats) |
| history messages do not count in the p95 latency | unit (match use case + stats) |
| a p95 over the target is marked | unit (console) |
| the model cannot keep up | unit (status loop with made-up ticks) |
| a queue that shrinks once gives no warning | unit (status loop) |
| status since start at stop | E2E |
| catch-up within its target | unit (console) |
| catch-up over its target | unit (console) |
| a non-local LLM is flagged | E2E |
| a local LLM gives no warning | E2E |
| peak load meets the speed target | E2E `@targets` |
| catch-up meets its time target | E2E `@targets` |
| a year of data fits | E2E `@targets` |
| a status interval below 10 seconds is refused | unit (parser) |
| defaults fill what the file leaves out | unit (parser) |
| every form of chat reference is accepted | unit (parser), unchanged |
| an invalid config stops the start | E2E (`settings.feature`), unchanged |
| every problem is listed at once | E2E (`settings.feature`), unchanged |
| a retry cap that is not a duration is refused | unit (parser), unchanged |

The `@targets` scenarios run in their own godog suite in the `e2e` module (`TestTargets`, concurrency 1, after `TestFeatures`, which skips `@targets`), so their timing is not shared with the other scenarios. The E2E module runs only in the full level of `scripts/qa.sh`, the before-merge check, so the three tests run only there; together they stay under a minute.

### D9. The load test

The fake LM Studio answers every call `yes` after 500 ms; the config has 4 workers, 100 chats and one request. The fake Telegram plays: 10 s of day flow at 20× the day's average (5,000/86,400 s × 20 ≈ 1.16 messages a second, spread over the 100 chats), then 10 s at 10 messages a second, then 10 s of day flow again. The test reads the sent file, where the fake now writes the time of each send, and the play time the fake writes per update into a played file; the p95 of send − play over all messages must be ≤ 10 s. Capacity is 8 calls a second, so the peak builds a queue of about 20 that drains in a few seconds: the expected p95 is about 3 s.

### D10. The catch-up test

The fake Telegram serves 100 chats × 500 history messages (ages inside the 3-day default depth); the fake LM Studio answers `no` at once; the config has 4 workers. The test waits for the catch-up summary and checks its time and the summary's own `done in` are at most 15 s (15 minutes scaled 60×) and that it judged 50,000. The fake's history read indexes the messages by chat once, so the fake does not add a quadratic cost.

### D11. The size test

The test starts tgscout once with no messages to create and migrate the store, stops it, then inserts 100,000 made-up judged messages (a `processed_message` row and a `decision` row each, the same columns and time format the store writes) in one transaction, checkpoints the WAL, and measures the file size growth per message. Projection: bytes per message × 10 requests × 5,000 × 365 < 10 GB. Counting the processed row 10 times makes the projection a bound above the real size.

## Risks / Trade-offs

- [The catch-up test may be slow on a busy machine: 50,000 messages each make a read and three writes in SQLite.] → It runs alone in its suite; the measured time on this machine is noted in tasks; if it is over budget, the fix is in tgscout's own write path, not in the target.
- [Load test timing on CI] → expected p95 ≈ 3 s against a 10 s target; the margin covers a slow runner.
- Measured on this machine (with `-race`, 2026-11-17): the load test runs 30 s with p95 2.7 s over 122 messages; the catch-up test gives its summary after 11.3 s (limit 15 s) with 50,000 judged; the size test runs 2 s and measures 153 bytes per judged message, 2.6 GB a year.
- [Telegram dates have whole seconds] → latency reads up to 1 s high, never low.

## Interface mock

[mock/index.html](mock/index.html) — the agreed console lines; this change keeps their content in the console's existing style.
