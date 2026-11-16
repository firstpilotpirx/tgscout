## Why

The design bends around numbers: a match reaches the owner within 10 s (p95) with a local LLM, for up to 100 chats, 5,000+ messages a day and a peak of 10 messages per second; catch-up ends within 15 minutes; the store stays under 10 GB a year; chat texts never leave the machine; the LLM costs 0 €. Without measuring them, version 1 cannot say it meets them, and the owner cannot see when the local model falls behind. This last slice of version 1 makes the numbers visible on the console and proves them with load tests.

Design pages this slice implements:

- [Quality attributes](../../../docs/architecture/quality-attributes.html) — every target, its unit, how it is measured and its priority; «the judge queue lag shows when the local model cannot keep up».
- [L1 System Context](../../../docs/architecture/context.html) — LM Studio is local; texts stay on the owner's Mac.
- [store — SQLite schema](../../../docs/architecture/store.html) — what grows per message and request.

## What Changes

- Each match line shows its latency: from the message's time in the chat to the confirmed send to the owner.
- Every few minutes, and at stop, the console shows one status line: messages in, judged, matches, queue length, p95 latency, p95 LLM call time; a warning when the queue keeps growing (the local model cannot keep up).
- The catch-up summary shows its time against the 15-minute target.
- A load test in `e2e/` with a fake Telegram and a fake LLM (with a set answer time) runs 100 chats, a peak of 10 messages per second, and a day's worth of 5,000 messages, and checks p95 ≤ 10 s from message to send.
- A catch-up test runs 100 chats × 500 messages and checks it ends within 15 minutes with the fake LLM's answer time.
- A size test fills the store with a year of decisions (5,000 messages a day × the requests of their chats) and checks the file stays under 10 GB.
- Privacy: at start tgscout warns when `lm_studio.url` is not on this machine (chat texts would leave it); the session file mode 600 check from `telegram-sign-in` stays; no message text goes into logs beyond the console match line.
- `README.md` gets a short «Targets» section: what the numbers are and how to read the status line.

Current stage: MVP. Target stage: MVP.
Scope: latency and queue metrics on the console, the status line, load, catch-up and size tests, the non-local LLM warning, the README targets section.
Acceptance bar: the three tests pass in `e2e/` (load, catch-up, size); a manual run shows the status line and latency per match with LM Studio.
Out of scope: metrics export (Prometheus, OpenTelemetry), dashboards, alerts; tuning the real model; a TUI or GUI (version 1.x and later).
Interface: CLI — the status line, latency on match lines, the start warning.

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

## Impact

- New: metrics in the core (latency, queue length), the status line printer, `e2e/` load, catch-up and size tests.
- Changed: match line format, catch-up summary, start checks, `README.md`.

## Open points

- The fake LLM's answer time in the tests: the p95 ≤ 10 s target depends on the real model. Proposed: 1 s per call with 4 workers in the load test, and the README states the real-model condition (a model that answers within about 2 s on the owner's Mac).
- Non-local LLM: warn or refuse. Proposed: warn, since the `llm` adapter is built to also reach OpenAI or OpenRouter when the owner chooses it.
- Status interval: proposed — 5 minutes, a setting.
- Load tests may be slow for `scripts/qa.sh`: proposed — they run in the before-merge level only, with a shortened day (time scaled) so they finish in under a minute.
- Version 1 done: after this slice, whether to tag `1.0.0` instead of `0.8.0` is the owner's call at its proposal.

## Release

SemVer: MINOR — targets visible and proven; the last slice of version 1.
Target version: 0.8.0.
