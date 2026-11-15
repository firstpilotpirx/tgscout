# Design progress — tgscout

Idea: finds what I need in Telegram chats — in new messages at once and in recent history — and notifies me through a Telegram bot. Language: Go. Board: http://192.168.100.151:8765/index.html.

Board format: 6

## Elements

Each element: `- [ ]` open, `- [ ] … — dialogue` items in progress, `- [ ] … — gate` waiting for acceptance, `- [x]` accepted; its pages follow as links. Nested as the C4 tree.

- [ ] L1 System Context — dialogue — [context](architecture/context.html)
- [ ] L2 Container — dialogue — [overview](architecture/overview.html)
- [ ] Quality attributes — [quality attributes](architecture/quality-attributes.html)

## To discuss

1. Peak 10 msg/s vs a judge of 1–3 s per message: one-by-one gives a growing queue at peak; batch several messages per request, run requests in parallel, or accept the lag at peak

## Later, not now

1. Images: check how well a local vision model reads a listing photo, then add photo search (video and audio stay out)
2. L3 Component of tgscout
3. Owner feedback (👍/👎, commands) on found messages

## Decisions

2026-11-15 — Design language: Go (board)
2026-11-15 — L1 drawn question by question, not from a story first (L1 System Context)
2026-11-15 — One actor in version 1: the owner; tgscout answers only the owner (L1 System Context)
2026-11-15 — Three external systems: Telegram over MTProto as a reader account, LM Studio as the local judge, Telegram Bot API for output (L1 System Context)
2026-11-15 — No Telegram Bot API, by design: a bot joins a group only when an admin adds it, and tgscout must join chats where the owner is not an admin. tgscout uses one ordinary Telegram account of its own (a second account, not the owner's) over MTProto both to read chats and to write matches to the owner; supersedes the line above (L1 System Context)
2026-11-15 — Every match also goes to the console, for debugging (L1 System Context)
2026-11-15 — The owner talks to tgscout in Telegram from the owner's own account, and runs the tgscout CLI and watches its console; tgscout is in Telegram as an ordinary user (L1 System Context)
2026-11-15 — Version 1 scope and non-goals as on the context page § About the system (L1 System Context)
2026-11-15 — Targets: ≤ 100 chats, 5,000+ messages/day, peak 10 msg/s, p95 ≤ 10 s to the owner, catch-up 500 messages or 3 days per chat by default (config) in ≤ 15 min, 0 missed and 0 repeated after restart, ≤ 10 GB/year, texts stay local, LLM cost 0 € (Quality attributes)
2026-11-15 — Containers: one Go CLI process, a SQLite store file and a matches.jsonl file; no broker, no separate services (L2 Container)
2026-11-15 — Requests: many standing requests, each bound to its own set of chats (1..N); every new message and every catch-up message of those chats goes to the LLM judge with that request; no text search, no embeddings in version 1 (L1 System Context)
2026-11-15 — Config format: YAML, the usual Go config format and easy to read (L2 Container)
2026-11-15 — Config tgscout.yaml holds requests with their chats, catch-up depth and LM Studio settings; settings are hierarchical: root, then request, then chat overrides; SQLite keeps only processed IDs and decisions (L2 Container)
2026-11-15 — No matches.jsonl: found messages are the «yes» decisions in SQLite; the only storage is SQLite (L2 Container)

## Review

## Plan

The slice order is drawn in `docs/index.html` § Plan as a `gantt` or `pipeline` block.

| Slice | Change | Release | Depends on |
|---|---|---|---|
2026-11-15 — Version 1 matches text only; images are later, after a check of a local vision model (L1 System Context)
2026-11-15 — Version 1 has no owner feedback and no commands: every found message goes to the owner as a plain private message (L1 System Context)
2026-11-15 — SQLite driver: mattn/go-sqlite3, the most used Go SQLite driver (9.3k GitHub stars) and the kit tech matrix choice; needs cgo (L2 Container)
