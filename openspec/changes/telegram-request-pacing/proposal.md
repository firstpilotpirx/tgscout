## Why

The owner worries that tgscout's own Telegram account gets limited or banned for too many requests. Live reading is a subscription (Telegram pushes updates over one connection, no request per message), so it is safe. Two places still send bursts of requests:

- Every start resolves every configured chat by name again. That call has a strict daily limit; with about 100 chats and several restarts a day tgscout comes close to it.
- Catch-up at start and after a reconnect pages `messages.getHistory` by 100 messages with no pause between calls: about 500 calls in a row for 100 chats × 500 messages. tgscout waits out each FLOOD_WAIT, but it should not provoke one, above all on a fresh account.

## What Changes

- tgscout remembers each resolved chat and the resolved owner in its database and does not resolve them by name again on the next start; it resolves again only when the config names a new chat or owner, or a stored one stops working.
- History reads of catch-up and of a reconnect gap keep a pause between calls, so the burst stays well under Telegram's limits.
- The console shows when a start reused stored chats and when catch-up is paced, in the existing style.

Current stage: MVP. Target stage: MVP.
Scope: chat and owner resolution at start and the pace of history calls.
Acceptance bar: tests prove a second start resolves no stored chat by name, and that history calls keep the set pause; FLOOD_WAIT handling stays as now.
Out of scope: changes to live reading (it is already a subscription); joining chats for the owner; sending to the owner.
Interface: CLI (console lines at start and during catch-up).

## Capabilities

### Modified Capabilities

- `telegram-session`: chats and the owner resolved once and stored.
- `catch-up`: a pause between history calls.
- `delivery`: the gap read after a reconnect uses the same pause.
- `settings`: the new root setting `history_pause`.

## Impact

- `tgscout/infra/telegram/` (chat and owner resolution, history paging, the fake Telegram of the E2E build), the SQLite schema (a new migration for stored chats), the catch-up flow.
- `docs/design-progress.md` (a new decision line) and `docs/architecture/store.html` (the new table).
- Specs, unit, integration and E2E tests of the capabilities above; `README.md`.

## Open points

- The pause between history calls: a fixed value or a setting. Decided: a root setting `history_pause`, default `1s`, `0s` turns it off.
- Whether the pause makes the 15-minute catch-up target miss at 100 chats × 500 messages (500 calls × 1 s ≈ 8 min). Decided: the default 1 s fits; the catch-up target test runs with the pause on.
- When a stored chat is resolved again: on a Telegram error about that chat, or also after some age. Decided: only on an error.
- Whether the owner is stored too. Decided by the owner at explore (2026-11-27, «Chats and owner»): yes; each start still checks that the owner's private chat is open.
- The 2026-11-15 decision «SQLite keeps only processed IDs and decisions» (`docs/design-progress.md`). Decided: replaced; the owner approved the design that keeps found chats and the owner in SQLite (2026-11-28, «Approve»). A new dated line in `docs/design-progress.md` records it.

## Release

SemVer: MINOR — new behavior and a new setting.
Target version: 0.9.0.
