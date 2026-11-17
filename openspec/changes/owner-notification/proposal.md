## Why

The owner should not have to watch a console: every match goes to the owner as a plain private Telegram message, written by tgscout's own account to the owner's own account, and also to the console for debugging. The send is part of the delivery promise: a message is marked processed only after Telegram confirmed the send, so no match is ever lost, and a rare repeat after a crash is allowed.

Design pages this slice implements:

- [L1 System Context](../../../docs/architecture/context.html) — matches reach the owner in a private chat with tgscout's account, and the console; no buttons, no feedback; tgscout writes only to the owner and only reads the watched chats.
- [Message path](../../../docs/flows/message-path.html) — «private message to owner + console → sent → only now: mark processed, move cursor».
- [L4 Code — matching](../../../docs/modules/matching/overview.html) — port `Notifier`, adapter `TelegramNotifierClient` (`secondary/notify/telegram`).
- [Quality attributes](../../../docs/architecture/quality-attributes.html) — Delivery: at least once, 0 lost matches.

## What Changes

- `TelegramNotifierClient` sends each match as a private message from tgscout's account to the owner through the shared MTProto client, and prints the same match line on the console.
- The message is plain text: the request id, the chat's title, the link to the message and its text (shortened when very long); no buttons.
- One match per request: a message that matches two requests gives two private messages.
- `MatchMessage` gets «sent» only after Telegram confirmed the send; only then the decision gets `sent_at` and, after the last request, the message is marked processed and the cursor moves.
- A send that fails leaves the message unprocessed; a FLOOD_WAIT from Telegram is waited out for the time Telegram names, then the send is tried again.
- tgscout never writes to a watched chat and never to anyone but the owner; private messages from other people, and the owner's own replies, are ignored (version 1 has no commands).

Current stage: PoC. Target stage: MVP.
Scope: the Telegram notifier, the message format, send-then-mark order, FLOOD_WAIT handling, writing only to the owner.
Acceptance bar: the scenarios below pass as slice and E2E tests with a faked Telegram send; a manual run with real accounts brings a match for a made-up message to the owner's Telegram within seconds.
Out of scope: owner feedback (👍/👎) and commands (later, not now); a summary message of catch-up (`catch-up`); retries of a long outage and crash tests (`delivery-resilience`).
Interface: Telegram private chat (the owner's own account) and the console.

## Acceptance scenarios

- Scenario: a match reaches the owner
  - Given a request bound to a watched chat and the LLM answers yes
  - When a new matching message appears in that chat
  - Then the owner gets one private message with the request, the chat and a link to the message
  - And the console shows the same match
- Scenario: two requests match one message
  - Given two requests of one chat and the LLM answers yes to both
  - When a new message appears in that chat
  - Then the owner gets two private messages, one per request
- Scenario: the message is marked processed only after the send
  - Given Telegram has not yet confirmed the send of a match
  - When tgscout stops at that moment
  - Then the message is not marked processed and the cursor has not moved past it
- Scenario: Telegram asks to wait
  - Given Telegram answers a send with a wait of 5 seconds
  - When tgscout sends a match
  - Then it waits 5 seconds, sends again and marks the message processed after the send is confirmed
- Scenario: a send fails
  - Given the send of a match fails
  - When the matcher handles the message
  - Then the message stays unprocessed and the console shows an error line
- Scenario: tgscout writes only to the owner
  - Given someone other than the owner writes to tgscout's account
  - When tgscout runs
  - Then tgscout does not answer and writes nothing to that person

## Capabilities

### New Capabilities

- `owner-notification`: sending each match to the owner as a private Telegram message and to the console, with the send confirmed before the message counts as processed.

### Modified Capabilities

- `message-matching`: a «yes» is sent to the owner, not only shown; the processed mark waits for the confirmed send; an edit is judged again where no «yes» was sent.
- `telegram-session`: the owner lookup also checks that the private chat with the owner is open.

## Impact

- New: `adapters/secondary/notify/telegram`.
- Changed: `MatchMessage` (send-then-mark with `sent_at`), `apps/cli/main.go` (wire the Telegram notifier instead of the console-only one).

## Open points

Settled by the preparation:

- The owner never wrote to tgscout's account: the README asks the owner to write «hi» to tgscout's account once and add it to contacts; at start tgscout checks the private chat with the owner is open and stops with «cannot write to the owner» if not.
- Length limit: the message text is cut to 1,000 characters with «…»; the link is always kept.
- Link for a chat with no username: the `t.me/c/<id>/<msg>` form.
- The private message names the request by its id and its text.
- FLOOD_WAIT: wait as long as Telegram says, no cap, then send again.
- The console shows the same text as the private message, under the match line.
- No «tgscout started» message.

## Release

SemVer: MINOR — matches reach the owner in Telegram.
Target version: 0.5.0.
