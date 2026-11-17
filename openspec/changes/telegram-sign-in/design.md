## Context

`walking-skeleton` and `hierarchical-settings` built the start flow up to the database: `apps/cli/start.go` loads `tgscout.yaml`, opens SQLite and waits for Ctrl+C. The [start flow](../../../docs/flows/start.html) puts «sign in as the reader account» right after SQLite. The [L3 page](../../../docs/architecture/tgscout.html) places the MTProto client and its session in `infra/telegram`, shared later by the collector (`live-matching`) and the notifier (`owner-notification`).

The preparation recorded every open point of the proposal: gotd, stop without a terminal, stop on a chat not found, wait on FLOOD_WAIT, a gotd session file with mode 600 and no extra encryption, the owner by `@username` or among the contacts, a faked client port and no network in tests, a hidden password, no login or logout command, and the approved mock.

## Goals / Non-Goals

**Goals:** one MTProto client per process; first interactive sign-in; silent later sign-in; a private session file; chat and owner lookup at start; no secret in the output; the README setup section.

**Non-Goals:** reading or sending messages, a reconnect after a network drop, a login or logout command, joining chats.

## Decisions

### 1. `infra/telegram` owns the client, the session and the start checks

`tgscout/infra/telegram/` holds:

- `Client` — the faked port: a small interface over gotd with `Run`, `Authorized`, `SendCode`, `SignIn`, `CheckPassword`, `Self`, `ResolveChat`, `FindOwner`. Errors it returns are the package's own (`ErrWrongCode`, `ErrWrongPassword`, `ErrPasswordNeeded`, `ErrSessionRevoked`, `FloodWait{Wait}`), never gotd's raw text, so nothing a server sends back (a phone number in an error, say) reaches the console.
- `SignIn(ctx, client, prompter, firstRun)` — the sign-in walk: authorized → done; first run → phone, code (again on a wrong code), password when needed (again on a wrong one).
- `Lookup(ctx, client, chats, owner, waits)` — resolves every chat, collects every one not found with its reason (`no such chat`, `the account has not joined it`), finds the owner; on `FloodWait` it reports the wait, sleeps (ctx-aware) and retries the same call. The sleep is a field so unit tests do not wait.
- `CheckSessionFile(path)` — absent, private, or open: a mode with any group or other bit (`mode & 0o077 != 0`) is open.
- `fileStorage` — gotd's `session.Storage` over the session file (the preparation's «gotd file session storage, mode 600, no extra encryption»: gotd's own session data in a plain file, mode 600; only the writer is ours). It loads the file when it exists. On a first run it keeps what gotd stores in memory until `Commit`, which `SignIn` calls after success; only then it writes the file through gotd's own `session.FileStorage`, which writes mode 600. gotd's own `FileStorage` would write the auth key before the sign-in succeeds, which the spec forbids («no session file SHALL be saved before the sign-in succeeds»). Decided at this run without a call (prepared change): a scope point the preparation did not settle, recommended option taken.
- `gotdClient` — the real `Client` over `github.com/gotd/td` v0.163.0 (pure Go, MIT, about 2.4k GitHub stars; released 2026-10-22, the newest release older than two weeks — `exact-versions.md`; its `go.mod` asks for Go 1.25.0, the project has 1.25.11): `auth.Status` for `Authorized`, `auth.SendCode` / `auth.SignIn` / `auth.Password`, `contacts.resolveUsername` for `@username` chats and owners, the account's dialogs for numeric chat IDs (a `-100` prefix is dropped), `contacts.getContacts` for a numeric owner. A 401 error (`AUTH_KEY_UNREGISTERED`, `SESSION_REVOKED`, `USER_DEACTIVATED`) and an unreadable session file map to `ErrSessionRevoked`; `tgerr.AsFloodWait` maps to `FloodWait`. gotd's logger stays the no-op default.
- `mapError(err)` — the pure mapping from gotd errors to the port's errors, unit-tested on gotd error values (`tgerr.New(401, "AUTH_KEY_UNREGISTERED")`, `FLOOD_WAIT_12`, `PHONE_CODE_INVALID`, `auth.ErrPasswordInvalid`), no network.
- `fakeClient` (build tag `telegramfake`) — reads a made-up scenario from the JSON file named by `TGSCOUT_FAKE_TELEGRAM`; `parseFakeScenario([]byte)` is unit-tested without the file system: whether the stored session is valid, the account name, the right code and password, the chats with their state, the contacts and a FLOOD_WAIT in seconds. `newClient` picks gotd without the tag and the fake with it; the production binary never holds the fake.

`infra/telegram` takes chats and the owner as plain strings, so `infra` does not import the matching component (`.go-arch-lint.yml`). Shared by later slices: the collector and the notifier get the same `Client` from `apps/cli`.

### 2. `apps/cli` wires it into the start

- `env.go` — `.env` in the current folder, parsed without a dependency (`KEY=VALUE`, `#` comments, optional quotes); a variable already in the environment wins. `TG_API_ID` must be a number; `TG_API_HASH` must be set; `TG_SESSION_FILE` defaults to `tgscout.session` next to the config (`*.session` is in `.gitignore`).
- Start order: config → database → credentials → session file check → `client.Run` → sign-in → lookup → «running» → wait for the stop; the rest of the start runs inside `client.Run`, which gotd needs. The stop closes the client by ending its context, then the database.
- `prompter.go` — the console prompts: phone and code read visibly from stdin, the password through `golang.org/x/term` v0.45.0 `ReadPassword` (hidden). `x/term` is a Go project sub-repository (its GitHub mirror has about 310 stars, below the matrix's 1,000-star bar); no hidden-input library with 1,000 or more stars exists apart from TUI frameworks far bigger than one prompt, so the risk is named and taken: it is maintained by the Go team with the toolchain. v0.45.0 (2026-07-08) is the newest release whose `go.mod` fits Go 1.25.11; v0.46.0 and later ask for Go 1.26. `term.IsTerminal(stdin)` decides whether a first run may ask; not a terminal → stop with «sign in once from a terminal first».
- The prompter, the terminal check and the client factory are fields of a `startDeps` struct, so unit tests drive the sign-in step — `connect(ctx, deps, con, chats, owner)`, which takes the client, the prompter and the console and reads no config file and opens no database — with a fake client and scripted input.

### 3. Console lines

The lines follow the [approved mock](mock/index.html) in the marks and wording of the existing console (`✓ ✗ ⚠ ● →`, `[ok]` … without color); the mock's «ERROR» word and its `● config:` line are not taken over, because `walking-skeleton` and `hierarchical-settings` already print `✗ <text>` and `✓ <path>: n requests …`:

```
● no session yet — first sign-in
Phone number: …
Telegram sent a login code to the account's app.
Login code: …
✗ the code is wrong — try again
Two-step password:
✓ signed in as <name>                       (· saved session on a later start)
✓ session saved <path> (mode 600)
⚠ Telegram asks to wait 12s (FLOOD_WAIT), then goes on
✓ owner found: <owner as written>
✓ watching <n> chats
✗ the session file is open to other users   hint: <path> has mode 644 · Fix: chmod 600 <path>
✗ Telegram no longer accepts the session    hint: delete <path>, then start tgscout from a terminal
✗ first sign-in needs a terminal            hint: sign in once from a terminal first, then run it as a service
✗ 2 chats not found                         rows: <chat>  <reason>; hint: join them with tgscout's account, or remove them from tgscout.yaml
✗ owner not found: <owner>                  hint: add the owner to the contacts of tgscout's account (README → Setup)
✗ TG_API_HASH is not set                    hint: fill it in .env (see .env.example)
```

The name is the account's first and last name; the phone is never printed, and the typed phone and code are not echoed back in any line.

### 4. Architecture lint

`.go-arch-lint.yml` gets the vendors `gotd` (`github.com/gotd/td/**`) and `xterm` (`golang.org/x/term`); `infra` may use `gotd`, `app` may use `xterm`.

### 5. E2E builds the fake

`scripts/qa.sh` builds the E2E binary with `-tags telegramfake`. The E2E world gives every start made-up `TG_API_ID`, `TG_API_HASH`, a session file path, a private made-up session file and a fake scenario where everything is found, so the `cli-lifecycle` and `settings` scenarios keep running; the sign-in steps change only what their scenario names.

This is an override, named here: the preparation's decision «A faked client port (own interface over gotd), no network in tests» replaces, for Telegram only, `testing-e2e-go.md` «External deps: Real or production-equivalent» and `testing-pyramid.md` «E2E … Built binary / script as in production. No mocks», and `testing-pyramid.md` «Every secondary adapter has a dedicated integration test … Real adapter + real infra»: `gotdClient` gets no integration test against real Telegram; the owner's manual run of the proposal's acceptance bar replaces it, and `mapError` is unit-tested. The binary is the production `main` with one adapter swapped at build time; config, database, `.env`, session file, console and the sign-in and lookup walks are the real ones.

The scenarios that type into tgscout (first sign-in, wrong code, secrets) run the binary on a pseudo-terminal (`github.com/creack/pty` v1.1.24, MIT, about 1.9k stars, 2024-10-31) in the `e2e` module, so every scenario of the spec is an E2E scenario of `e2e/features/telegram_session.feature`, one `Rule:` per requirement.

## Test levels

| Scenario | Level | Where |
|---|---|---|
| the api hash is missing | E2E | `e2e/features/telegram_session.feature` |
| first start signs in and saves the session | E2E | feature, on a pseudo-terminal with echo on: the typed phone shows, the password does not (hidden) |
| a wrong login code | E2E | feature, on a pseudo-terminal |
| first sign-in without a terminal | E2E | feature (stdin of the process is a pipe) |
| a later start signs in silently | E2E | feature |
| the session file is open to others | E2E | feature |
| a revoked session | E2E | feature |
| every chat and the owner are found | E2E | feature |
| a chat the account has not joined | E2E | feature |
| Telegram asks to wait during the lookup | E2E | feature (fake FLOOD_WAIT of 1 s) |
| the owner is not found | E2E | feature |
| secrets never reach the console | E2E | feature, on a pseudo-terminal |

Unit tests (no file system, no network) cover `SignIn`, `Lookup` (FLOOD_WAIT with a fake sleep), `mapError`, `parseFakeScenario`, the `.env` parser on a string, the mode check on a given `fs.FileMode` and the console lines; integration tests (`-tags=integration`) cover `CheckSessionFile` and `fileStorage` on a real temp folder (no file before `Commit`, mode 600 after). The real gotd network path is checked only by the owner's manual run (proposal § Acceptance bar); `manual_run` in `.claude/kit.yaml` lists no area, so the run does not stop for it.

## Risks / Trade-offs

- [gotd's real network path is not covered by automated tests] → thin adapter, every branch maps to the port's errors; the manual run of the acceptance bar covers it.
- [A build tag for the fake] → the fake lives in one file under `//go:build telegramfake`; the plain build fails to compile if it references the fake.
- [Lookup of numeric chat IDs reads all dialogs] → one paged read at start, at most a few hundred dialogs for a PoC account.

## Interface mock

[Telegram sign-in mock](mock/index.html) — first start, later start and start errors, dark and light; approved at preparation.

## Release

MINOR, 0.2.0 → 0.3.0, bumped in the last task group.
