## 1. Session file and storage

- [x] 1.1 Red: unit tests for the mode check on a given `fs.FileMode` (600 private; 644, 640 open, the mode in the error); integration tests for `CheckSessionFile` on a temp folder (absent, private, open)
- [x] 1.2 Green: `infra/telegram/session_file.go`
- [x] 1.3 Red: integration tests for `fileStorage` — loads an existing file; on a first run writes nothing before `Commit`, mode 600 after
- [x] 1.4 Green: `infra/telegram/file_storage.go` (memory until `Commit`, then gotd's `session.FileStorage`, mode 600); add gotd v0.163.0 to `go.mod`
- [x] 1.5 Commit (`kit ship`)

## 2. Sign-in and lookup walks

- [x] 2.1 Red: unit tests for `SignIn` with a fake `Client` and prompter — authorized session asks nothing; first run asks phone, code, password; wrong code and wrong password ask again; `Commit` only after success; revoked session error
- [x] 2.2 Green: `infra/telegram/client.go` (port, errors), `infra/telegram/sign_in.go`
- [x] 2.3 Red: unit tests for `Lookup` — every chat found; not joined and missing chats listed together with reasons; owner not found; FLOOD_WAIT reported, slept (fake sleep) and retried
- [x] 2.4 Green: `infra/telegram/lookup.go`
- [x] 2.5 Commit (`kit ship`)

## 3. Real and fake clients

- [x] 3.1 Red: unit tests for `mapError` on gotd error values — 401 errors → revoked, `FLOOD_WAIT_12` → `FloodWait{12s}`, `PHONE_CODE_INVALID` → wrong code, invalid password → wrong password, other errors → a fixed text without the server's message
- [x] 3.2 Green: `infra/telegram/gotd_errors.go`; `infra/telegram/gotd_client.go` over gotd (auth flow, resolveUsername, dialogs, contacts, no-op logger) and `newClient` without the tag — the network path is checked by the owner's manual run
- [x] 3.3 Red: unit test (tag `telegramfake`) for `parseFakeScenario` on a made-up JSON string
- [x] 3.4 Green: `infra/telegram/fake_client.go` under `//go:build telegramfake`
- [x] 3.5 `.go-arch-lint.yml`: vendors `gotd`, `xterm`; `infra` may use `gotd`, `app` may use `xterm`
- [x] 3.6 Commit (`kit ship`)

## 4. Start flow in the CLI

- [x] 4.1 Red: unit tests for the `.env` parser on a string and for credentials — comments, quotes, environment wins, missing `TG_API_HASH`, non-numeric `TG_API_ID`, default session path next to the config
- [x] 4.2 Green: `apps/cli/env.go`
- [x] 4.3 Red: unit tests for the new console lines (signed in, session saved, wait, owner found, watching, chats-not-found list)
- [x] 4.4 Green: console lines in `apps/cli/console_telegram.go`
- [x] 4.5 Red: unit tests for `connect` (the sign-in step: client, prompter, console; no config file, no database) with a fake client and scripted input — first run prints the sign-in lines in order, a wrong code asks again, a lookup failure stops with exit 1
- [x] 4.6 Green: `apps/cli/prompter.go` (hidden password via `golang.org/x/term` v0.45.0, terminal check), `startDeps`, sign-in and lookup inside `client.Run` in `apps/cli/start.go`; update existing start tests for the new deps
- [x] 4.7 Commit (`kit ship`)

## 5. E2E

- [x] 5.1 Red: `e2e/features/telegram_session.feature` (`@telegram_session @e2e`) with every scenario of the spec, one `Rule:` per requirement, and their steps; the typing scenarios run the binary on a pseudo-terminal (`github.com/creack/pty` v1.1.24); the world gives every start made-up credentials, a private session file and a fake scenario
- [x] 5.2 Green: `scripts/qa.sh` builds the E2E binary with `-tags telegramfake`; fix what the scenarios find
- [x] 5.3 Commit (`kit ship`)

## 6. Docs and version

- [x] 6.1 `README.md` § Setup: get `api_id` / `api_hash` at my.telegram.org, prepare the second account, join the chats with it, add the owner to its contacts, first start from a terminal; `.env.example` comments for the three variables
- [x] 6.2 Bump `VERSION` to 0.3.0
- [ ] 6.3 Commit (`kit ship`)

Manual checks by the user: none for this run (`manual_run` lists no area); the owner's real-account sign-in of the acceptance bar stays for the owner.
