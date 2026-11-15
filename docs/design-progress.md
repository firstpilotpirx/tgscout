# Design progress — tgscout

Idea: finds what I need in Telegram chats — in new messages at once and in recent history — and notifies me through a Telegram bot. Language: Go. Board: http://192.168.100.151:8765/index.html.

Board format: 6

## Elements

Each element: `- [ ]` open, `- [ ] … — dialogue` items in progress, `- [ ] … — gate` waiting for acceptance, `- [x]` accepted; its pages follow as links. Nested as the C4 tree.

- [ ] L1 System Context — dialogue — [context](architecture/context.html)
- [ ] L2 Container — [overview](architecture/overview.html)

## To discuss

1. L1 System Context — who uses tgscout and which external systems it talks to

## Later, not now

1. Nothing postponed yet

## Decisions

2026-11-15 — Design language: Go (board)
2026-11-15 — L1 drawn question by question, not from a story first (L1 System Context)

## Review

## Plan

The slice order is drawn in `docs/index.html` § Plan as a `gantt` or `pipeline` block.

| Slice | Change | Release | Depends on |
|---|---|---|---|
