## Why

tgscout does not run all the time: while it is off, messages keep coming. At every start it must read the history it has not seen yet — from the cursor of each chat, up to a set depth (by default 500 messages or 3 days per chat, overridable per request and per chat) — and judge it the same way as new messages, while it already listens to new ones. A restart must repeat nothing and lose nothing.

Design pages this slice implements:

- [Catch-up flow](../../../docs/flows/catch-up.html) — chats one after another, messages oldest first from the cursor, a processed message is skipped, each new one goes through the message path, then the summary on the console.
- [Start flow](../../../docs/flows/start.html) — catch-up and real-time listening run at once after the workers start.
- [store — SQLite schema](../../../docs/architecture/store.html) — `chat_cursor` read at start, `processed_message` read to skip.
- [L2 Container § Config](../../../docs/architecture/overview.html) and [L3 § Data structures](../../../docs/architecture/tgscout.html) — catch-up depth at root, request and chat.
- [Quality attributes](../../../docs/architecture/quality-attributes.html) — catch-up 500 messages or 3 days per chat, ≤ 15 min from start to the summary.

## What Changes

- At start, after the workers start, the collector runs catch-up beside live listening.
- Catch-up takes the watched chats one after another; for each it reads history over MTProto from the chat's cursor towards now, within the effective depth, and hands the messages oldest first to `CollectMessage`, the same path as new messages (pre-filter, processed check, queue, matcher, notifier).
- A chat with no cursor yet (first start) reads the last depth window.
- Each message is judged only against the requests of its chat whose depth covers it (a chat in two requests with different depths reads up to the larger depth).
- A message that is already processed is skipped, so a restart repeats nothing; a message that arrives live while its chat is being caught up is judged once.
- When every chat is read and every caught-up message is processed, tgscout prints the summary on the console: chats read, messages read, skipped, judged, matches, and the time it took.
- Telegram's FLOOD_WAIT during history reads is waited out and the read goes on.

Current stage: MVP. Target stage: MVP.
Scope: history read at start from the cursor within the effective depth, the depth rules, running beside live listening, the summary line.
Acceptance bar: the scenarios below pass as slice tests with a fake Telegram history; a manual run after tgscout was off for a while finds a made-up matching message posted meanwhile and prints the summary.
Out of scope: deep history search with a summary (version 2); measuring the 15-minute target on 100 chats (`quality-targets`); reconnect gaps while running (`delivery-resilience`).
Interface: CLI — the summary line; matches go to the owner as in live matching.

## Acceptance scenarios

- Scenario: first start reads the default depth
  - Given a watched chat with no cursor and 4 days of history
  - When the owner starts tgscout
  - Then tgscout judges the messages of the last 3 days, at most 500, oldest first
- Scenario: a restart reads only what it missed
  - Given a chat whose cursor is at message 100 and 20 newer messages
  - When the owner starts tgscout
  - Then only the 20 newer messages are judged
- Scenario: a processed message is not judged again
  - Given a message after the cursor that is already marked processed
  - When catch-up reaches it
  - Then it is skipped and no second match is sent
- Scenario: a request overrides the depth
  - Given a request with a catch-up depth of 7 days
  - When catch-up reads a chat of that request for the first time
  - Then it reads back 7 days for that request
- Scenario: a chat overrides the depth
  - Given a chat of a request with a catch-up depth of 2,000 messages
  - When catch-up reads that chat for the first time
  - Then it reads up to 2,000 messages
- Scenario: a match in the history reaches the owner
  - Given a matching message posted while tgscout was off
  - When the owner starts tgscout
  - Then the owner gets the match as a private message
- Scenario: live and catch-up at once
  - Given catch-up is reading a chat
  - When a new message appears in that chat
  - Then that message is judged once and the owner gets at most one match per request for it
- Scenario: the summary
  - Given three watched chats
  - When catch-up has read all of them and every caught-up message is processed
  - Then the console shows the summary with chats, messages, matches and the time

## Capabilities

### New Capabilities

- `catch-up`: reading the missed history of every watched chat at start, within its depth, through the same message path, ending with a summary.

## Impact

- Changed: `adapters/primary/collect/telegram` (history reader), `CollectMessage` (requests filtered by depth), `apps/cli/main.go` (the fork of the start flow).

## Open points

- Where history matches go: the context page says «history matches come as one summary message», the decisions say every found message goes to the owner as a plain private message, and the catch-up flow prints the summary on the console. Proposed: each history match is its own private message, as in live matching, and the summary goes to the console only; the context page line is fixed on the board in this change.
- Both limits set: proposed — stop at whichever limit comes first (500 messages or 3 days, whichever is fewer messages).
- Order of chats: proposed — as listed in `tgscout.yaml`.
- A cursor older than the depth window (tgscout off for weeks): proposed — read only the depth window and say in the summary how many chats had a gap.

## Release

SemVer: MINOR — tgscout catches up on missed history.
Target version: 0.6.0.
