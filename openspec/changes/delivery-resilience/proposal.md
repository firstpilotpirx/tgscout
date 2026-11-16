## Why

The design promises at-least-once delivery: no match is ever lost, a rare repeat after a crash is allowed, and after a restart nothing is missed and nothing is repeated for messages that were fully processed. The earlier slices put the order in place (send, then mark processed, then move the cursor). This slice makes the promise hold when things go wrong: LM Studio stops or is slow, Telegram drops the connection, the owner stops tgscout in the middle of work, or the process is killed. It also adds the crash test the quality attributes name.

Design pages this slice implements:

- [Quality attributes](../../../docs/architecture/quality-attributes.html) — Delivery: at least once, 0 lost matches, rare repeats allowed; «crash test: kill between send and mark → the match comes again, never lost»; 0 missed and 0 repeated after restart.
- [Message path](../../../docs/flows/message-path.html) — mark processed and move the cursor only after every «yes» was sent.
- [L2 Container decisions](../../../docs/architecture/overview.html) — at peak messages wait in the in-memory queue; after a restart catch-up rereads from the cursor and skips processed IDs, so nothing is lost.
- [Start flow](../../../docs/flows/start.html) — listening runs until the owner stops tgscout.

## What Changes

- LLM failures: a failed or timed-out call is retried with growing pauses (for example 1 s, 2 s, 4 s … up to a cap); while LM Studio is down the workers wait, the queue fills, and the collector waits when the queue is full; the console shows one line when LM Studio goes away and one when it is back.
- Telegram send failures: retried the same way until sent; the message stays unprocessed meanwhile.
- Telegram connection drops: tgscout reconnects by itself; after a reconnect it reads each chat again from its cursor (a short catch-up), so messages from the gap are judged.
- Stop on Ctrl+C or SIGTERM: the collector stops taking messages, each worker finishes the message it holds within a grace time, nothing unfinished is marked processed, the database is closed, exit code 0. A second Ctrl+C stops at once.
- Crash: messages in the in-memory queue are lost from memory but not from Telegram; the next start's catch-up rereads them from the cursor. A match that was sent but not yet marked comes again (the allowed rare repeat).
- An end-to-end crash test kills the process between the send and the mark and checks the match comes again after restart; another kills it with a full queue and checks no message is lost.

Current stage: MVP. Target stage: MVP.
Scope: retries with backoff for the LLM and the send, back-pressure when the queue is full, reconnect with a gap catch-up, graceful stop, crash tests.
Acceptance bar: the scenarios below pass; the two crash tests pass in `e2e/` against fakes of Telegram and LM Studio.
Out of scope: a persistent queue on disk (the cursor and catch-up replace it, by design); running as a system service; alerts outside the console.
Interface: CLI — status and error lines on the console.

## Acceptance scenarios

- Scenario: LM Studio goes away and comes back
  - Given LM Studio stops answering while messages arrive
  - When LM Studio answers again
  - Then every waiting message is judged and every match reaches the owner, none lost
- Scenario: the queue is full
  - Given LM Studio is down long enough that the queue fills up
  - When more new messages arrive
  - Then the collector waits, and after LM Studio is back every message is judged or is read again by catch-up
- Scenario: the connection to Telegram drops
  - Given tgscout runs and the connection to Telegram drops for a while
  - When the connection is back
  - Then the messages posted during the drop are judged
- Scenario: the owner stops tgscout during work
  - Given a worker is judging a message
  - When the owner presses Ctrl+C
  - Then the worker finishes that message or leaves it unprocessed, and tgscout exits with code 0
- Scenario: a crash between send and mark
  - Given a match was sent to the owner and the message was not yet marked processed
  - When the process is killed and the owner starts tgscout again
  - Then the owner gets that match again, and no match is lost
- Scenario: a crash with messages in the queue
  - Given messages are waiting in the queue
  - When the process is killed and the owner starts tgscout again
  - Then every one of those messages is judged after the restart
- Scenario: a clean restart repeats nothing
  - Given every message read so far is processed
  - When the owner stops and starts tgscout
  - Then no match is sent again

## Capabilities

### New Capabilities

- `delivery`: the at-least-once promise under failures — retries, back-pressure, reconnect, graceful stop and crash recovery.

## Impact

- Changed: `OpenAiLlmClient`, `TelegramNotifierClient`, `ChannelMessageQueue`, the collector (reconnect and gap read), `apps/cli/main.go` (signals and grace time).
- New: crash tests in `e2e/`.

## Open points

- Retry cap and grace time: proposed — backoff up to 60 s, grace time 10 s; both as settings in `tgscout.yaml` with these defaults.
- A message that fails with the LLM forever (for example too long for the model): proposed — after 10 failed tries save «no» with a warning, so one bad message cannot stall its chat's cursor.
- Gap read after reconnect: the MTProto library's own update-gap recovery, or our cursor read. Proposed: our cursor read, the same code as catch-up, so there is one way to recover.

## Release

SemVer: MINOR — delivery holds under failures.
Target version: 0.7.0.
