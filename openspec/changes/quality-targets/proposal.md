## Why

The design bends around numbers: a match reaches the owner within 10 s (p95) with a local LLM, for up to 100 chats, 5,000+ messages a day and a peak of 10 messages per second; catch-up ends within 15 minutes; the store stays under 10 GB a year; chat texts never leave the machine; the LLM costs 0 €. Without measuring them, version 1 cannot say it meets them, and the owner cannot see when the local model falls behind. This last slice of version 1 makes the numbers visible on the console and proves them with load tests.

Design pages this slice implements:

- [Quality attributes](../../../docs/architecture/quality-attributes.html) — every target, its unit, how it is measured and its priority; «the judge queue lag shows when the local model cannot keep up».
- [L1 System Context](../../../docs/architecture/context.html) — LM Studio is local; texts stay on the owner's Mac.
- [store — SQLite schema](../../../docs/architecture/store.html) — what grows per message and request.

## What Changes

- Each match line shows its latency: from the message's time in the chat to the confirmed send to the owner.
- Every `status_interval` (a new root setting in `tgscout.yaml`, default `5m`, at least `10s`), and at stop, the console shows one status line: messages in, judged, matches, queue length, p95 latency and p95 LLM call time over the last interval (the stop line covers the whole run), each p95 marked against its target; a warning when the queue length grew in 3 status lines in a row (the local model cannot keep up).
- The catch-up summary shows its time against the 15-minute target.
- A load test in `e2e/` with a fake Telegram and a fake LLM answering in 0.5 s with 4 workers runs 100 chats and a peak burst of 10 messages per second for 10 s inside a day's flow, and checks p95 ≤ 10 s from message to send; the day's flow runs at its average rate (5,000 a day) scaled up 20 times, 10 s before and 10 s after the peak.
- A catch-up test runs 100 chats × 500 messages with time scaled 60 times (15 minutes → 15 s; the fake LLM answers at once, so the test proves tgscout's own work fits) and checks the summary comes within the scaled 15 minutes.
- A size test fills the store with 100,000 made-up decisions, measures bytes per row and projects a year (5,000 messages a day × the requests of their chats) under 10 GB.
- Privacy: at start tgscout warns (and keeps running) when the host of `lm_studio.url` is not `localhost`, `127.0.0.1` or `::1` (chat texts would leave it); the session file mode 600 check from `telegram-sign-in` stays; no message text goes into logs beyond the console match line.
- `README.md` gets a short «Targets» section: what the numbers are, how to read the status line, and the real-model condition: p95 ≤ 10 s needs a model that answers within about 2 s, and with a local model the 15-minute catch-up holds only for about 2,000 judged messages.
- The load, catch-up and size tests run only in the before-merge level of `scripts/qa.sh`; together they take under a minute.
- Interface mock: [mock/index.html](mock/index.html), agreed at preparation.

Current stage: MVP. Target stage: MVP.
Scope: latency and queue metrics on the console, the status line and its `status_interval` setting, load, catch-up and size tests, the non-local LLM warning, the README targets section.
Acceptance bar: the three tests pass in `e2e/` (load, catch-up, size); a manual run shows the status line and latency per match with LM Studio.
Out of scope: metrics export (Prometheus, OpenTelemetry), dashboards, alerts; tuning the real model; a TUI or GUI (version 1.x and later).
Interface: CLI — the status line, latency on match lines, the catch-up target mark, the start warning; config — the `status_interval` key.

## Acceptance scenarios

- Scenario: latency on a match line
  - Given a match
  - When the owner gets it
  - Then the console match line shows the time from the message to the send
- Scenario: the status line
  - Given tgscout runs for a while
  - When the status interval passes
  - Then the console shows messages in, matches, queue length and p95 latency
- Scenario: the model cannot keep up
  - Given the queue keeps growing for several status intervals
  - When the next status line is printed
  - Then it carries a warning that the model is behind
- Scenario: peak load meets the speed target
  - Given 100 watched chats and a peak of 10 messages per second with a fake LLM
  - When the load test runs
  - Then p95 from message to send is at most 10 seconds
- Scenario: catch-up meets its time target
  - Given 100 chats with 500 missed messages each and a fake LLM
  - When tgscout starts
  - Then the catch-up summary comes within 15 minutes
- Scenario: a year of data fits
  - Given a store filled with a year of decisions at 5,000 messages a day
  - When its size is measured
  - Then it is under 10 GB
- Scenario: a non-local LLM is flagged
  - Given `lm_studio.url` points to another machine
  - When the owner starts tgscout
  - Then tgscout warns that chat texts will leave this machine

## Capabilities

### New Capabilities

- `quality-targets`: the version 1 targets made visible on the console and proven by load, catch-up and size tests.

### Modified Capabilities

- `settings`: the new root key `status_interval` (default 5 minutes, at least 10 seconds) and its check.

## Impact

- New: metrics in the core (latency, queue length), the status line printer, `e2e/` load, catch-up and size tests.
- Changed: match line format, catch-up summary, start checks, `README.md`.

## Decided at preparation

- Fake LLM in the load test: 0.5 s per call, 4 workers, a 10 s peak burst at 10 messages per second.
- Catch-up test: time scaled 60 times; the README states the real-model limit.
- Heavy tests: before-merge level only, under a minute together.
- Non-local LLM: warn and keep running.
- Status interval: setting `status_interval`, default 5 minutes; p95 over the last interval.
- «Model is behind»: the queue grew in 3 status lines in a row.
- Size test: a 100,000-row sample projected to a year.
- Version: 0.8.0; 1.0.0 comes later, after the owner's manual run.

## Decided by the prepared run

- The mock's content (latency, status block, behind warning, target marks, start warning, stop status) is kept in the console's existing style: status marks and indented detail lines, no clock time or INFO/WARN prefix, so every line stays like the lines before it.
- The day's flow of the load test: its average rate (5,000 a day) scaled up 20 times, played 10 s before and 10 s after the peak; the full 5,000 messages cannot pass a 0.5 s LLM with 4 workers in under a minute (at least 625 s).
- The made-up LLM of the catch-up test answers at once: 0.5 s scaled 60 times would still need over 100 s for 50,000 calls; the test proves tgscout's own work for 50,000 messages fits the scaled 15 s, and the README states the real-model limit.
- Size projection: bytes per judged message with one decision, times 10 requests per message (a bound above the real mix), times 5,000 a day for a year.

## Release

SemVer: MINOR — targets visible and proven; the last slice of version 1.
Target version: 0.8.0.
