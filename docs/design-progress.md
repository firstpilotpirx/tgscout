# Design progress — tgscout

Idea: finds what I need in Telegram chats — in new messages at once and in recent history — and notifies me through a Telegram bot. Language: Go. Board: http://192.168.100.151:8765/index.html.

Board format: 6

## Elements

Each element: `- [ ]` open, `- [ ] … — dialogue` items in progress, `- [ ] … — gate` waiting for acceptance, `- [x]` accepted; its pages follow as links. Nested as the C4 tree.

- [ ] L1 System Context — dialogue — [context](architecture/context.html)
- [ ] L2 Container — [overview](architecture/overview.html)

## To discuss

1. L1 System Context — what tgscout does and does not do (scope and non-goals)
2. How the owner gives 👍/👎 and commands without bot buttons (an ordinary account cannot send inline buttons)

## Later, not now

1. Nothing postponed yet

## Decisions

2026-11-15 — Design language: Go (board)
2026-11-15 — L1 drawn question by question, not from a story first (L1 System Context)
2026-11-15 — One actor in version 1: the owner; tgscout answers only the owner (L1 System Context)
2026-11-15 — Three external systems: Telegram over MTProto as a reader account, LM Studio as the local judge, Telegram Bot API for output (L1 System Context)
2026-11-15 — No Telegram Bot API, by design: a bot joins a group only when an admin adds it, and tgscout must join chats where the owner is not an admin. tgscout uses one ordinary Telegram account of its own (a second account, not the owner's) over MTProto both to read chats and to write matches to the owner; supersedes the line above (L1 System Context)
2026-11-15 — Every match also goes to the console, for debugging (L1 System Context)

## Review

## Plan

The slice order is drawn in `docs/index.html` § Plan as a `gantt` or `pipeline` block.

| Slice | Change | Release | Depends on |
|---|---|---|---|
