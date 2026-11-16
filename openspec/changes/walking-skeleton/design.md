## Context

See proposal.md § Why. The repository holds only the design board and OpenSpec documents; no Go code, no checks. The kit rules for Go (`architecture-go.md`, `tech-matrix-go.md`, `testing-*-go.md`) and the folders picture on the L3 page fix the layout. The preparation of this change recorded the user's answers: plain `tgscout` with flags on one cobra root command, the database next to the config file, golang-migrate with embedded SQL, godog with `.feature` files under `e2e/features/`, viper for the YAML file, no CI and no `--version` flag.

## Goals / Non-Goals

**Goals:**

- A `tgscout` binary that walks the first three steps of the start flow (load config → valid? → open SQLite) and stops cleanly.
- The checks every later slice reuses: `scripts/qa.sh`, `.golangci.yml`, `.go-arch-lint.yml`, the godog runner in `e2e/`.

**Non-Goals:**

- Reading any field of `tgscout.yaml` (`hierarchical-settings` adds the schema and may replace the YAML library).
- Package folders with no code yet: Go has no empty packages, so `core/domain`, `adapters/primary/*`, `adapters/secondary/{queue,llm,notify}` and `infra/telegram` appear with the slice that fills them.
- A GitHub Actions workflow (recorded answer: keep the scope).

## Decisions

### Layout

```
tgscout/                                   # workspace: go.work (use .) + go.mod module github.com/firstpilotpirx/tgscout
  apps/cli/                                # package main
    main.go                                # os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    root_command.go                        # cobra root command, flags, exit codes
    start.go                               # the start flow: config → database → wait for a signal → stop
    console.go                             # ✓ ✗ ● → lines, color and [ok]/[fail] fallback
  components/matching/
    matching_config.go                     # MatchingConfig: wires the use cases from the ports
    core/application/
      collect_message.go, match_message.go # use cases, no behavior yet
      settings_repository.go, message_repository.go, message_queue.go, llm.go, notifier.go   # driven ports
    adapters/secondary/
      settings/yaml/yaml_config_file.go    # CheckConfigFile(path): exists? valid YAML? (viper)
      persistence/sqlite/
        sqlite_migrator.go                 # Migrate(db): golang-migrate over iofs, reports created / up to date
        migrations/000001_init.{up,down}.sql
  infra/sqlite/sqlite_database.go          # Open(path) *sql.DB with mattn/go-sqlite3, foreign keys and WAL on
e2e/                                       # module github.com/firstpilotpirx/tgscout/e2e, no replace of the app module
  features/cli_lifecycle.feature
  cli_lifecycle_test.go                    # godog steps, drives the built binary from SUT_BIN
scripts/qa.sh
```

- Ports are interfaces with no methods yet and the use cases are structs holding their ports; each later slice adds the methods it calls. Alternative — guess the methods now — rejected: the slices that use them (`live-matching`, `owner-notification`) settle their shapes.
- `MatchingConfig` takes a `Ports` struct; `main` passes no adapters yet, the first slice that adds one passes it there. Only `apps/cli` imports adapters and `infra`.
- The yaml check sits in `secondary/settings/yaml`, where `YamlSettingsRepository` will live, so `hierarchical-settings` replaces the file's content without moving it.

### Config check (viper)

`CheckConfigFile` runs `os.Stat` first: a missing path gives `ErrConfigNotFound`; then `viper.SetConfigFile` + `ReadInConfig`. A parse error is turned into `ConfigSyntaxError{Path, Line, Reason}`: the line number is taken from the YAML error text (`line N: …`). Alternative — a direct YAML library — rejected for now by the recorded answer (viper, as the tech matrix says); `hierarchical-settings` may change it.

### Database and migrations

- `infra/sqlite.Open` opens the file with the DSN `file:<path>?_foreign_keys=on&_journal_mode=WAL` and pings it.
- Created vs up to date: `start.go` checks with `os.Stat` whether the file existed before opening it; `Migrate` returns the schema version and the number of the app's tables (from `sqlite_master`, without `schema_migrations` and `sqlite_*`).
- golang-migrate with the `sqlite3` database driver (mattn-based) and the `iofs` source over `//go:embed migrations/*.sql`; `ErrNoChange` means up to date. Alternative — own runner — rejected by the recorded answer.
- The first migration creates the three tables of the store page: `chat_cursor (chat_id PK, last_message_id, updated_at)`, `processed_message (chat_id, message_id, processed_at; PK both ids)`, `decision (chat_id, message_id, request_id, matched, decided_at, sent_at NULL; PK the three ids)`. No foreign keys between them (the store page draws the links dashed).
- An open or migrate error gives exit code 1 and a `✗ database …` line naming the path.

### Command, exit codes and stop

- cobra root command `tgscout`, `Args: cobra.NoArgs`, flags `--config` (default `./tgscout.yaml`), `--db` (empty: `tgscout.db` in `filepath.Dir(config)`), `-h/--help` with the usage text «show this help»; `SilenceErrors` and `SilenceUsage` on, so `run` decides the output.
- `run` returns the exit code: 0 after a clean stop or `--help`; 1 for a config or database error; 2 for a flag or argument error (wrapped by `SetFlagErrorFunc` and the `Args` check into one `usageError`), printed as `✗ <error>` plus the usage to stderr.
- `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`; after `● running — press Ctrl+C to stop`, it waits for the context, prints `→ stopping…`, closes the database and prints `✓ stopped`.
- Output follows the agreed mock: success lines on stdout, `✗` lines and their hint on stderr.

### Console colors

`fatih/color`, the one color library. `color.NoColor` already covers a non-terminal stdout, `NO_COLOR` and `TERM=dumb`; `FORCE_COLOR=1` sets it back to false. When `NoColor` is true the marks fall back to `[ok]`, `[fail]`, `[run]` and `->`. Palette: green ✓, red ✗ and the offending path, cyan ● and paths, dim hints.

### Test levels

| Spec scenario | Level | Where |
|---|---|---|
| the config file is missing | E2E | `e2e/features/cli_lifecycle.feature` |
| the config file is not valid YAML | E2E | same |
| a first start creates the database | E2E | same |
| a second start keeps the database | E2E | same |
| tgscout starts and stops cleanly | E2E | same |
| tgscout stops on a terminate signal | E2E | same |
| the owner asks for help | E2E | same |
| the owner gives an unknown option | E2E | same |

- Every scenario is CLI behavior seen from outside (exit code, console, the file), so the E2E level drives the real binary with `os/exec`; there is no primary adapter yet, so no slice test.
- Below them: unit tests for every file of `core/application`, `matching_config.go` and `console.go`; `run` tested in `apps/cli` with temp dirs; integration tests (`//go:build integration`, real SQLite and real files in `t.TempDir()`) for `yaml_config_file.go`, `sqlite_migrator.go` and `infra/sqlite`.
- Gherkin runner: godog v0.16.0 called from `go test` (`godog.TestSuite`), `.feature` files under `e2e/features/`, tags `@cli_lifecycle @e2e`; `@wip` excluded. The e2e module reads the database with `mattn/go-sqlite3` itself; it imports no app source.

### Checks

- `scripts/qa.sh [quick|full]`: `quick` runs `go vet`, `golangci-lint run`, `go-arch-lint check`, `go mod tidy -diff` and `go test -race -coverprofile ./...` in `tgscout/`; `full` (default) adds `govulncheck ./...`, `lizard --csv` with `scripts/check-complexity-ratchet.sh` against `metrics/cyclomatic.tsv` (R-136), `typos` (`_typos.toml`), `shellcheck scripts/*.sh .githooks/*`, `jscpd` (`.jscpd.json`, Go sources), `go test -race -tags=integration ./...`, builds the binary into a temp dir and runs `e2e/` against it with `SUT_BIN`, then `gitleaks git`. `apidiff` and `perlcritic` do not apply (no library module, no Perl). `.claude/kit.yaml` gets `after_group: scripts/qa.sh quick` and `before_merge: scripts/qa.sh`.
- `.golangci.yml` is the tech matrix's lint configuration as is.
- `.go-arch-lint.yml` (version 3): components `app` (`apps/**`), `infra` (`infra/**`), `config` (`components/matching` root), `application` (`core/application`), `domain` (`core/domain/**`), `adapters` (`adapters/**`); `application` may depend on `domain` only, `adapters` on `application` and `domain`, `config` on `application`, `app` on everything; vendors cobra, viper, migrate, sqlite3, color allowed only where they are used.
- Pinned tools (R-139): golangci-lint 2.15.0, go-arch-lint v1.19.1, govulncheck v1.9.0, lizard 1.24.2, typos 1.51.1, shellcheck 0.11.0, jscpd 5.4.2, gitleaks 8.30.1. A `require_tool` function in `qa.sh` fails when a tool is missing or at another version and prints its install command; it installs nothing.
- `.githooks/pre-commit` runs `gitleaks git --redact`; `README.md` § Install says to run `git config core.hooksPath .githooks` once, and `qa.sh` warns when `core.hooksPath` is not set.

### Versions

Go 1.25.11 in `go.mod` (the floor golang-migrate v4.20.1 sets; the tech matrix asks 1.23+) and `toolchain go1.27.4` in `go.work` and both `go.mod` files (`govulncheck` finds GO-2026-6859 in the 1.27.3 standard library; the go command downloads the pinned toolchain), cobra v1.10.2, viper v1.21.0, migrate v4.20.1, go-sqlite3 v1.14.52 (v1.14.53 is under two weeks old), godog v0.16.0, testify v1.12.1, color v1.19.0.

## Interface mock

[mock/cli.html](mock/cli.html) — the five console frames (start and Ctrl+C, missing config, bad YAML, second start, help), agreed at preparation.

## Risks / Trade-offs

- [cgo for mattn/go-sqlite3] → a C compiler is needed to build; README says so. No CI in this change.
- [The line number comes from the YAML error text] → an integration test pins the format; `hierarchical-settings` switches to a library with typed positions.
- [Empty ports look like `any`] → each is a named interface with a doc comment saying which slice adds its methods; none is used as a value type yet.
- [E2E signal tests are timing-dependent] → the step waits for the `running` line before sending the signal, with a 10 s timeout.

## Migration Plan

None: first code. Rollback is reverting the merge.
