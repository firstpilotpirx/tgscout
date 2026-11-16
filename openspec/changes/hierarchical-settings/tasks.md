# Tasks

## 1. Domain types and the port

- [x] 1.1 Red: unit tests for `NewChatRef` (`@username`, `t.me` link with and without `https://`, numeric ID, negative numeric ID accepted; empty, spaces, a short username, another host rejected; a link and `@name` of one username share one `Key()`), `NewOwner` (no link) and `Settings.DistinctChats`; verify: `go test -race ./components/matching/core/domain/...` fails to compile
- [x] 1.2 Green: `core/domain` types of design § Domain types and `SettingsRepository.Settings() domain.Settings`; verify: the 1.1 tests pass
- [x] 1.3 Commit the group (`kit ship`)

## 2. Parse and resolve tgscout.yaml

- [x] 2.1 Red: unit tests for `yaml.Parse`: the five catch-up scenarios of the `settings` spec (root, request override, chat override, one chat in two requests, built-in defaults), the defaults of workers and LM Studio, and the three forms of chat reference; verify: `go test -race -run TestParse ./components/matching/adapters/secondary/settings/yaml/...` fails
- [x] 2.2 Green: `Parse` walking the goccy/go-yaml AST and resolving chat → request → root → default; add `github.com/goccy/go-yaml v1.19.2`; verify: the 2.1 tests pass
- [x] 2.3 Red: unit tests for `Parse` errors, each with its line and field: no requests, a request without id, find or chats, a duplicate id (naming the first line), an unknown key at root, request, chat and catch-up level, a wrong kind of value, a zero and a negative depth, a bad `lm_studio.url`, no model, `workers` 0, a missing and a malformed owner, a malformed chat reference, `api_id` / `api_hash` / `session` at any level (not also an unknown key), several errors at once sorted by line; and the warning for 101 distinct chats (none for 100); verify: they fail
- [x] 2.4 Green: the validation and the warning in `Parse`; verify: the 2.3 tests pass
- [x] 2.5 Commit the group (`kit ship`)

## 3. Load the file and replace the viper check

- [x] 3.1 Red: integration tests for `LoadSettings`: a missing path gives `ErrConfigNotFound`; broken YAML on line 4 gives `ConfigSyntaxError` with line 4; an invalid file gives `ConfigErrors` with its path; a valid file gives the resolved settings and its warnings; verify: `go test -race -tags=integration -run TestLoadSettings ./components/...` fails
- [x] 3.2 Green: `LoadSettings` and `YamlSettingsRepository`; delete `CheckConfigFile` and its tests; drop viper from `go.mod`, `go.sum` and `.go-arch-lint.yml`, add goccy to the adapters' vendors; verify: the 3.1 tests pass and `go mod tidy -diff` prints nothing
- [x] 3.3 Commit the group (`kit ship`)

## 4. Start: summary, errors and warning on the console

- [ ] 4.1 Red: unit tests for `console.warn` (`⚠`, `[warn]` when plain) and for the error list format of the mock (`✗ <path>: 3 errors`, `1 error`, aligned `line <n>  <field>  <reason>` lines); verify: they fail
- [ ] 4.2 Green: `console.warn`, the error list printer and `checkConfig` using `LoadSettings`: summary line, warnings, errors; pass the repository to `matching.Ports.Settings`; verify: the 4.1 tests pass
- [ ] 4.3 Red: update the fixtures of `apps/cli` tests to a full valid config; integration tests for `run`: an invalid config gives 1, every error and no database; a valid config prints `[ok] <path>: 1 requests…`; verify: the new tests fail
- [ ] 4.4 Green: wire it in `start.go` so the 4.3 tests pass; verify: `go test -race -tags=integration ./apps/...` passes
- [ ] 4.5 Commit the group (`kit ship`)

## 5. E2E scenarios

- [ ] 5.1 Red: `e2e/features/settings.feature` (`@settings @e2e`) with the scenarios of the `settings` spec at E2E level (the invalid-config outline, every problem at once, too many chats, the summary), their steps, and the full valid config in the `cli_lifecycle` steps; verify: `SUT_BIN=<old binary> go test -race -count=1 ./...` in `e2e/` fails on the new scenarios
- [ ] 5.2 Green: build the new binary; verify: `SUT_BIN=<new binary> go test -race -count=1 ./...` passes
- [ ] 5.3 Commit the group (`kit ship`)

## 6. Example, docs and version

- [ ] 6.1 `tgscout.yaml.example` with every field and made-up values; add `tgscout.yaml` to `.gitignore`; README config section; the L2 board sample in `docs/architecture/overview.html` with the L3 names; `owner` in the L3 `tgscout.yaml` entity picture of `docs/architecture/tgscout.html`; verify: `tgscout --config tgscout.yaml.example` prints the summary line
- [ ] 6.2 Bump `VERSION` to 0.2.0; verify: `kit version-check hierarchical-settings` passes
- [ ] 6.3 Commit the group (`kit ship`)

## Manual checks

None.
