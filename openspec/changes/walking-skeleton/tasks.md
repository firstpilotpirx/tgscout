# Tasks

## 1. Workspace, ports and checks

- [x] 1.1 Create `tgscout/go.work` (`use .`) and `tgscout/go.mod` (module `github.com/firstpilotpirx/tgscout`, Go 1.25.11) with the pinned versions of design § Versions; verify: `go mod tidy -diff` prints nothing
- [x] 1.2 Red: unit tests for `NewCollectMessage`, `NewMatchMessage` and `NewMatchingConfig` keeping their ports; verify: they fail to compile (types missing)
- [x] 1.3 Green: the five method-less driven ports, the two use cases in `core/application` and `MatchingConfig` with its `Ports` struct; verify: `go test -race ./components/...` passes
- [x] 1.4 Add `.golangci.yml` (tech matrix lint configuration), `.go-arch-lint.yml`, `_typos.toml`, `.jscpd.json`, `metrics/cyclomatic.tsv`, `scripts/check-complexity-ratchet.sh` and `scripts/qa.sh [quick|full]` with the pinned-version guard (design § Checks); set `after_group` and `before_merge` in `.claude/kit.yaml`; verify: `scripts/qa.sh quick` passes and a wrong pinned version makes it fail
- [x] 1.5 Add `.githooks/pre-commit` running `gitleaks git --redact` and set `git config core.hooksPath .githooks`; verify: the next `kit ship` runs the hook
- [x] 1.6 Commit the group (`kit ship`)

## 2. Config file check

- [x] 2.1 Red: integration tests for `CheckConfigFile`: a missing path gives `ErrConfigNotFound`; broken YAML on line 4 gives `ConfigSyntaxError` with line 4; a valid file gives no error; verify: `go test -race -tags=integration -run TestCheckConfigFile ./components/...` fails
- [x] 2.2 Green: `secondary/settings/yaml/yaml_config_file.go` with viper; verify: the 2.1 tests pass
- [x] 2.3 Commit the group (`kit ship`)

## 3. Database and first migration

- [x] 3.1 Red: integration tests for `infra/sqlite.Open` (opens a new file in a temp dir, pings it; a path in a missing folder gives an error naming it) and for `Migrate` (a new file gets version 1 and the three tables; a second run reports up to date and keeps a row); verify: they fail
- [x] 3.2 Green: `infra/sqlite/sqlite_database.go`, `persistence/sqlite/sqlite_migrator.go` and `migrations/000001_init.{up,down}.sql` with the store schema; verify: the 3.1 tests pass
- [x] 3.3 Commit the group (`kit ship`)

## 4. The tgscout command

- [x] 4.1 Red: unit tests for `console.go`: marks and colors, and `[ok]` / `[fail]` when color is off; verify: they fail
- [x] 4.2 Green: `apps/cli/console.go` with `fatih/color` and the `FORCE_COLOR` override; verify: the 4.1 tests pass
- [x] 4.3 Red: tests for `run` in temp dirs: `--help` gives 0 and the usage; an unknown flag or a stray argument gives 2 and the usage on stderr; a missing config gives 1 and no database; bad YAML gives 1 and line 4; a valid config creates `tgscout.db` beside it, says `created`, and a cancelled context gives 0 and `stopped`; a second start says `schema up to date`; verify: they fail
- [x] 4.4 Green: `apps/cli/main.go`, `root_command.go` and `start.go` as design § Command; verify: the 4.3 tests pass
- [x] 4.5 Commit the group (`kit ship`)

## 5. E2E with godog

- [ ] 5.1 Red: `e2e/go.mod` (godog, testify, go-sqlite3; no `replace` of the app module), `e2e/features/cli_lifecycle.feature` with the eight spec scenarios tagged `@cli_lifecycle @e2e`, step definitions in `e2e/cli_lifecycle_test.go` driving `SUT_BIN`; run against a binary that does not exist yet in the e2e path; verify: the run fails
- [ ] 5.2 Green: build the binary, `SUT_BIN=<bin> go test ./...` in `e2e/`; fix any gap the scenarios show in `apps/cli`; verify: all eight scenarios pass
- [ ] 5.3 Commit the group (`kit ship`)

## 6. Docs, examples and version

- [ ] 6.1 Add `.env.example` (empty `TG_API_ID`, `TG_API_HASH`, `TG_SESSION_FILE`) and `tgscout.yaml.example` (made-up requests and chats, valid YAML); verify: `tgscout --config tgscout.yaml.example --db <tmp>` starts
- [ ] 6.2 `README.md`: sections `## Install` (Go, a C compiler for cgo), `## Run (dev)`, `## Run (production)`, `## Test` (`scripts/qa.sh`); verify: the four headings exist
- [ ] 6.3 `AGENTS.md`: § Gherkin names godog and `e2e/features/`; § How to verify a change names `scripts/qa.sh quick|full` and the Red/Green command; the layout line points to `tgscout/`; verify: read back
- [ ] 6.4 Bump `VERSION` to 0.1.0; verify: `kit version-check walking-skeleton` passes
- [ ] 6.5 Commit the group (`kit ship`)

## Manual checks

- none (the owner's manual start with the example config is covered by the E2E scenarios; `.claude/kit.yaml` lists no manual-run area)
