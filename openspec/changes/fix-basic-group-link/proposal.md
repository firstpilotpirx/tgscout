## Why

The private match message tgscout sends to the owner holds a link to the found message. For a basic group (an old-style, non-super group) the link has the form `https://t.me/c/<chat id>/<message id>`. Telegram opens `t.me/c/…` links only for supergroups and channels, so for a basic group the link leads nowhere. Found in the owner's first manual run: a test basic group gave a link like `https://t.me/c/5400000000/7` (made-up ID) that does not open.

## What Changes

- For a match from a basic group, the owner no longer gets a broken `t.me/c/…` link.
- Supergroups (IDs starting with `-100`), channels and public chats keep their links as now.
- The console match line follows the same rule as the Telegram message.

Current stage: MVP. Target stage: MVP.
Scope: the message link in the owner's match message and in the console match line.
Acceptance bar: a test proves a basic-group match has no `t.me/c/…` link, and supergroup and public-chat matches keep theirs.
Out of scope: links for private chats with one person; changes to matching or delivery.
Interface: CLI (the console match line) and the Telegram message to the owner.

## Capabilities

### Modified Capabilities

- `owner-notification`: the link line of a match from a basic group.
- `message-matching`: the console match scenario promises the link only when the chat has one.

## Impact

- The code that builds the match message text and its link.
- The owner-notification and message-matching specs, their tests and `.feature` scenarios.

## Open points

- What to show instead of the link for a basic group: drop the link line, or a short line such as "no link: basic group". Proposed: drop the link line.
- Whether a working link exists for a basic group at all (for example a link that opens the chat, not the message). Proposed: check at design; if none, drop the line.

## Release

SemVer: PATCH — a bug fix of an existing behavior.
Target version: 0.8.1.
