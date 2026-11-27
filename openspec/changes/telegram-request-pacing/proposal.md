## Why

The owner worries that tgscout's own Telegram account gets limited or banned for too many requests. Live reading is a subscription (Telegram pushes updates over one connection, no request per message), so it is safe. Two places still send bursts of requests:

- Every start resolves every configured chat by name again. That call has a strict daily limit; with about 100 chats and several restarts a day tgscout comes close to it.
- Catch-up at start and after a reconnect pages `messages.getHistory` by 100 messages with no pause between calls: about 500 calls in a row for 100 chats × 500 messages. tgscout waits out each FLOOD_WAIT, but it should not provoke one, above all on a fresh account.

## What Changes

- tgscout remembers each resolved chat in its database and does not resolve it by name again on the next start; it resolves again only when the config names a new chat or a stored one stops working.
- History reads of catch-up and of a reconnect gap keep a pause between calls, so the burst stays well under Telegram's limits.
- The console shows when a start reused stored chats and when catch-up is paced, in the existing style.

Current stage: MVP. Target stage: MVP.
Scope: chat resolution at start and the pace of history calls.
Acceptance bar: tests prove a second start resolves no stored chat by name, and that history calls keep the set pause; FLOOD_WAIT handling stays as now.
Out of scope: changes to live reading (it is already a subscription); joining chats for the owner; sending to the owner.
Interface: CLI (console lines at start and during catch-up).

## Capabilities

### Modified Capabilities

- `telegram-session`: chats resolved once and stored.
- `catch-up`: a pause between history calls.
- `delivery`: the gap read after a reconnect uses the same pause.

## Impact

- `tgscout/infra/telegram/` (chat resolution, history paging), the SQLite schema (a new migration for stored chats), the catch-up flow.
- Specs, unit, integration and E2E tests of the capabilities above; `README.md`.

## Open points

- The pause between history calls: a fixed value or a setting. Proposed: a root setting `history_pause`, default `1s`.
- Whether the pause makes the 15-minute catch-up target miss at 100 chats × 500 messages (500 calls × 1 s ≈ 8 min). Proposed: default 1 s fits; check at design.
- When a stored chat is resolved again: on a Telegram error about that chat, or also after some age. Proposed: only on an error.

## Release

SemVer: MINOR — new behavior and a new setting.
Target version: 0.9.0.
