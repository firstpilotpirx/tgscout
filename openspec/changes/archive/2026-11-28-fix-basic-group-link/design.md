## Context

See proposal.md — Why. The collector builds the link from the chat's username: with one it is `t.me/<username>/<id>`, without one always `t.me/c/<id>/<id>`. The start already knows each chat's Telegram type (a `tg.Chat` is a basic group, a `tg.Channel` is a supergroup or a channel), but the type does not reach the collector. Basic groups never have a username, so only the no-username branch is wrong.

## Goals / Non-Goals

**Goals:** no dead link for a basic group; the same text on the console and in Telegram.

**Non-Goals:** links for private chats with one person; a link that opens the group itself.

## Decisions

| # | Question | Chosen | Why | Rejected |
|---|---|---|---|---|
| D1 | What the owner gets for a basic group | no link line; the other lines stay | Telegram has no message link for a basic group at all; a note line adds noise | a «no link: basic group» line; a link to the chat (needs an invite link the account may not have) |
| D2 | How tgscout knows a chat is a basic group | `ChatInfo` gets `Basic bool`, set from the Telegram type the start finds: a dialog peer `InputPeerChat` or a resolved `tg.Chat`; it flows into the collector's `Chat` | the type is the fact; the reference text can be written either way | guess from the config text (`-…` vs `-100…`): the owner may write the bare ID |
| D3 | Where the link line is dropped | the collector gives an empty `Link`; `NotificationText` leaves out the link line when `Link` is empty | one rule in the domain text; the console prints the same text, so it follows on its own | a separate console rule |
| D4 | The fake client | a numeric reference that starts with `-` but not `-100` is a basic group | lets the E2E scenario drive a basic group with made-up IDs | a new scenario field |

## Test levels

| Scenario | Level | Why |
|---|---|---|
| a chat without a username | unit (collector) | as before |
| a basic group has no link line | E2E (`owner_notification.feature`) plus unit tests of the collector and of `NotificationText` | the owner sees the whole message; the E2E drives the built binary with the fake client |
| a long message text is cut | unit, as before | unchanged |
| a match reaches the owner | E2E, as before | only the THEN wording changes |
| two requests match one message | E2E, as before | unchanged |
| a matching message is shown on the console | E2E, as before | only the THEN wording changes |
| a message that does not match | E2E, as before | unchanged |

## Interface mock

[Basic group match mock](mock/index.html) — the Telegram message and the console match for a basic group, before and after, in a dark and a light terminal.

## Risks / Trade-offs

- [A dialog peer type the code does not know] → `Basic` stays false and the link stays as today.

## Release

PATCH, 0.8.0 → 0.8.1, bumped in the last task group.
