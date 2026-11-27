## Context

See proposal.md § Why. Today `telegram.Lookup.Find` (`infra/telegram/lookup.go`) calls `Client.ResolveChat` for every config chat and `Client.FindOwner` for the owner at every start. The gotd client keeps each found chat's input peer in memory only (`gotdClient.peers`), so `ReadHistory` works only for chats resolved in this run. `@name` chats and an `@name` owner cost one `contacts.resolveUsername` each (the call with a strict daily limit); a numeric chat ID pages `messages.getDialogs` over all of the account's chats. `pageHistory` (`history.go`) calls `messages.getHistory` page after page with no pause; catch-up (`HistoryCollector.CatchUp`) and the reconnect gap read (`HistoryCollector.GapRead`) go through it, one chat after another. The SQLite file is open before the Telegram start (`apps/cli/start.go`), its schema lives in the matching component's migrations (`000001_init`).

## Goals / Non-Goals

**Goals:** a restart sends no name lookup for a stored chat or owner; every `messages.getHistory` call of the process starts at least `history_pause` after the one before; the console says both.

**Non-Goals:** pacing other calls (the sends to the owner, the owner's open-chat check, the updates); a store age or a refresh timer; live reading.

## Decisions

### D1. One table `known_peer` in a new migration

`000002_known_peer` adds:

```sql
CREATE TABLE known_peer (
    role        TEXT NOT NULL,      -- 'chat' or 'owner'
    ref         TEXT NOT NULL,      -- the config text, as the config writes it
    kind        TEXT NOT NULL,      -- 'channel', 'chat' or 'user'
    peer_id     INTEGER NOT NULL,
    access_hash INTEGER NOT NULL,   -- 0 for a basic group
    username    TEXT NOT NULL,
    title       TEXT NOT NULL,
    resolved_at TIMESTAMP NOT NULL,
    PRIMARY KEY (role, ref)
);
```

The key is the config text: a changed spelling is a new entry and is looked up once. The table holds no message text; the access hash is a per-account value Telegram gives for the peer, it stays in the local database, never in the repository (it is like the session file). Alternative: one table per role. Rejected: same columns, twice the code.

### D2. The store is a port of `infra/telegram`, kept by the SQLite adapter

`infra/telegram` defines `Peer{Kind, ID, AccessHash}`, `StoredPeer{Peer, Username, Title}` and

```go
type PeerStore interface {
    Load(ctx context.Context, role Role, ref string) (StoredPeer, bool, error)
    Save(ctx context.Context, role Role, ref string, peer StoredPeer) error
    Forget(ctx context.Context, role Role, ref string) error
}
```

`components/matching/adapters/secondary/persistence/sqlite.SqlitePeerRepository` implements it over `known_peer`; the app wires it into `telegram.Lookup`. `Lookup.Store` nil means no store (the tests that do not care, as today). Alternative: put the table and its code in `infra/sqlite`. Rejected: the migrations live in the matching component's adapter; two migration sets on one file would fight over `schema_migrations`.

### D3. The client port gains peer data and two calls without network

- `ChatInfo` gets `Peer Peer`; `ResolveChat` fills it (`*tg.Channel` → `channel` with its access hash, `*tg.Chat` → `chat`, a numeric ID from the dialog's input peer).
- `KeepChat(info ChatInfo)` keeps a stored chat's input peer for `ReadHistory`; no call to Telegram.
- `FindOwner` returns `OwnerInfo{State, Peer}` instead of `OwnerState`.
- `CheckOwner(ctx, owner string, peer Peer) (OwnerState, error)` keeps the stored owner's peer for `SendToOwner` and asks only whether the private chat is open (`messages.getPeerDialogs`, the call `FindOwner` makes today after the lookup).

The fake client of the E2E build implements the same calls (D7).

### D4. `Lookup.Find` uses the store first

For each chat: a stored entry → `KeepChat`, counted as stored; else `ResolveChat` (with `Retry`), and a joined chat is saved. Missing and not-joined chats are reported as today and never saved. For the owner: a stored entry → `CheckOwner`; when it is not `OwnerReady`, `Forget` and fall back to `FindOwner` (the spec's «drop, find again, check once more»); a found owner is saved, an owner that is `OwnerMissing` is not. `Find` returns the found chats and `LookupCounts{Stored, LookedUp}`; `console.watching(chats, counts)` prints `watching 4 chats · 3 from the store, 1 looked up` (only the parts that are not zero; see the mock). A store error stops the start like a database error (`database …: …`).

### D5. A stored chat that stops working

`mapError` maps `CHANNEL_INVALID`, `CHANNEL_PRIVATE`, `CHAT_ID_INVALID` and `PEER_ID_INVALID` to a new `ErrChatGone`. `Lookup.Refresh(ctx, ref string, id int64) (ChatInfo, error)` forgets the stored chat, resolves it by name (with `Retry`), and saves and returns it when it is joined under the same ID; otherwise it returns `ErrChatGone`. `HistoryCollector` gets a `Refresher` (the `Lookup`; nil keeps today's behavior): on `ErrChatGone` from `ReadHistory` it calls `Refresh` once, reports `OnRefreshed(chat)`, and reads again; a second failure fails the read as any failed read does (the cursor stays, `next start reads it again`). The chat is already gone from the store then, so the next start looks it up by name and reports «not joined» or «no such chat» through the normal lookup. Console: `⚠ catch-up: <name> · Telegram no longer knows the stored chat · looked it up again` (gap read: `gap read: …`).

Alternative: forget the chat and leave the lookup to the next start. Rejected: the chat would stay unread for the whole run.

### D6. `telegram.Pacer` spaces the history calls

`Pacer{Pause; now; sleep}` with a mutex and the start time of the last call: `Wait(ctx)` sleeps until `last + Pause`, then sets `last = now`. The pause runs between call starts, so a slow call uses up part of it. One pacer per client, made from `Config.HistoryPause`; the gotd `ReadHistory` calls `Wait` before each `messages.getHistory` page, so catch-up, the gap read and a repeat after FLOOD_WAIT all share it, also when a gap read starts while catch-up still runs. `Pause` 0 returns at once. The fake client uses the same pacer.

Catch-up target: 100 chats × 500 messages is 500 calls (5 pages of 100 per chat), so 500 × 1 s ≈ 8.3 min of pacing, within the 15-minute target with the judge working in parallel. The scaled catch-up target test (60×) sets `history_pause: 17ms` (1 s scaled) so it proves the target with the pause on.

Alternative: pace in `HistoryCollector` per chat. Rejected: a chat of 500 messages is 5 calls in a burst, and the gap read and catch-up could still overlap.

### D7. `history_pause` setting and the fake Telegram

- YAML root key `history_pause`, a duration, default `1s`; below 0 is the field error `must be 0s or more`. `domain.Settings.HistoryPause`; `openTelegram` passes it into `telegram.Config`.
- `catchUpStarted(chats, pause)` appends ` · 1s between history reads` when the pause is not 0.
- The fake scenario gets `calls_file` (each `resolve` and `history` call appended as a JSON line with its kind, chat and Unix-nano time), and `gone_chats` (config texts whose first history read answers `ErrChatGone`). The fake keeps no store of its own: the real SQLite store runs in the E2E, so a second start proves no `resolve` line.

### D8. Test levels

| Spec scenario | Level |
|---|---|
| a restart looks up no stored chat by name | E2E (`telegram_session.feature`, two starts, `calls_file`) |
| a chat added to the config is looked up | unit (`Lookup` with a made-up store and client) |
| Telegram no longer knows a stored chat | E2E (`gone_chats`) |
| the account left a stored chat | unit (`HistoryCollector` + `Lookup.Refresh`) |
| every chat and the owner are found | E2E (existing) |
| a chat the account has not joined | E2E (existing) |
| Telegram asks to wait during the lookup | E2E (existing) |
| the owner is not found | E2E (existing) |
| the owner never wrote to tgscout's account | E2E (existing) |
| a stored owner whose chat was closed | unit (`Lookup`) |
| history calls keep the pause | E2E (`catch_up.feature`, `calls_file`) and unit (`Pacer` with a made-up clock) |
| no pause when it is off | unit (`Pacer`) |
| one line per chat | E2E (existing) |
| the start line names the pause | unit (console) |
| the gap read keeps the pause | unit (`Pacer` shared by two readers; the gap read uses the same client) |
| the connection to Telegram drops | E2E (existing) |
| a message processed during the drop is not judged twice | E2E (existing) |
| defaults fill what the file leaves out | unit (YAML parser) |
| every form of chat reference is accepted | unit (existing) |
| a negative history pause is an error | unit (YAML parser) |

The SQLite repository gets an integration test (save, load, forget, the migration to version 2).

## Risks / Trade-offs

- [A stored chat the account left stays «found» at start until a history read fails] → the first catch-up read fails or refreshes it (D5); live updates of a left chat do not come anyway.
- [A username moved to another chat] → `Refresh` checks the ID; a different ID fails the read and the next start looks it up, then the owner sees the new chat's title.
- [The scaled catch-up target test gets slower by ~8 s of pacing] → it stays under its 15 s; if apply measures more, the pause stays and the cause is reported.
- [The access hash in the local database] → same trust as the session file next to it; nothing leaves the machine.

## Migration Plan

Migration `000002` only adds a table; an old database gets it at the next start, the first start after the update looks up every chat once. Rollback: the down step drops the table.

## Interface mock

[Console lines: lookup counts, the paced catch-up start line, the re-lookup line](mock/index.html) — light and dark terminals.
