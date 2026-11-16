## Why

tgscout has an accepted design (L1–L4 on the design board) and no code yet. Every later slice of version 1 needs the same base: the Go workspace laid out as the design's folders picture says, a `tgscout` program that starts, reads its config file, opens its SQLite file with the schema, and stops cleanly. The walking skeleton builds exactly that base and the project checks around it, so each next slice adds one use case end to end instead of setting up the repository again.

Design pages this slice implements:

- [L2 Container](../../../docs/architecture/overview.html) — one Go CLI process, `tgscout.yaml`, one SQLite file; no broker.
- [store — SQLite schema](../../../docs/architecture/store.html) — tables `chat_cursor`, `processed_message`, `decision`.
- [L3 Component — tgscout § Folders and layers](../../../docs/architecture/tgscout.html) — repository and workspace layout, hexagon of `matching`.
- [L4 Code — matching](../../../docs/modules/matching/overview.html) — use cases, ports and adapters (here only their empty packages and the wiring).
- [Start flow](../../../docs/flows/start.html) — the steps «load tgscout.yaml → config valid? → open SQLite»; sign-in and workers come in later slices.

## What Changes

- The repository gets the layout of the folders picture: `tgscout/` workspace (`go.work` + one `go.mod`), `tgscout/apps/cli/`, `tgscout/components/matching/core/application`, `tgscout/components/matching/adapters/secondary/{settings/yaml,persistence/sqlite}`, `tgscout/infra/sqlite`, `e2e/` with its own `go.mod`, `scripts/qa.sh`. Folders with no code yet (`core/domain`, the primary adapters, `secondary/{queue,llm,notify}`, `infra/telegram`) come with the slice that fills them: Go has no empty packages.
- `tgscout` starts from the console: it reads `tgscout.yaml` (path from a flag, default `./tgscout.yaml`), prints a clear error and exits with a non-zero code when the file is missing or is not valid YAML.
- It opens the SQLite file `tgscout.db` through `mattn/go-sqlite3` (`infra/sqlite`) and applies the first migration: the three tables of the store schema. A second start reuses the file and changes nothing.
- It stops cleanly on Ctrl+C (SIGINT) and SIGTERM: closes the database and exits with code 0.
- `MatchingConfig` wires the one component; the use cases `CollectMessage` and `MatchMessage` and the five driven ports (`SettingsRepository`, `MessageRepository`, `MessageQueue`, `Llm`, `Notifier`) exist as types with no behavior yet.
- `scripts/qa.sh` runs the Go static analysis stack of the tech matrix (`go vet`, `golangci-lint`, `govulncheck`, `go mod tidy -diff`, coverage, `lizard` with its ratchet, `typos`, `shellcheck`, `jscpd`), the tests and `gitleaks git`; every tool is pinned and checked for its version; an architecture check (`.go-arch-lint.yml`) keeps the hexagon's dependencies pointing inwards.
- A committed pre-commit hook (`.githooks/pre-commit`) runs `gitleaks git` before every commit, as `AGENTS.md` § Open source and secrets asks.
- The Gherkin runner is chosen and wired (`AGENTS.md` § Gherkin says the first code change decides it), with the folder for `.feature` files.
- `.env.example` (empty `TG_API_ID`, `TG_API_HASH`, session path) and `tgscout.yaml.example` (made-up requests and chats) are added; `README.md` says how to build and run.
- `AGENTS.md` names the real check commands and the layout once they exist.

Current stage: PoC. Target stage: PoC.
Scope: the Go workspace and folders, the CLI start and stop, reading the config file as YAML, opening SQLite with the first migration, the architecture check, `scripts/qa.sh`, the Gherkin runner, the example files.
Acceptance bar: `scripts/qa.sh` passes; the scenarios below pass as tests; `tgscout` started by hand with the example config creates `tgscout.db` with the three tables and stops on Ctrl+C with exit code 0.
Out of scope: the settings schema and its hierarchy (next slice, `hierarchical-settings`); Telegram sign-in; reading messages; the LLM; sending to the owner; catch-up.
Interface: CLI — the `tgscout` command, its console output and exit codes.

## Acceptance scenarios

- Scenario: tgscout starts and stops cleanly
  - Given a valid config file and no database file yet
  - When the owner starts tgscout and then presses Ctrl+C
  - Then tgscout creates the database with the cursor, processed message and decision tables
  - And it exits with code 0
- Scenario: the config file is missing
  - Given no config file at the given path
  - When the owner starts tgscout
  - Then tgscout prints an error that names the missing path
  - And it exits with a non-zero code without creating the database
- Scenario: the config file is not valid YAML
  - Given a config file with broken YAML
  - When the owner starts tgscout
  - Then tgscout prints an error with the line of the problem
  - And it exits with a non-zero code
- Scenario: a second start keeps the database
  - Given a database created by an earlier start, with rows in it
  - When the owner starts tgscout again
  - Then the rows are still there and the schema is unchanged

## Capabilities

### New Capabilities

- `cli-lifecycle`: how the `tgscout` command starts, opens its stores and stops.

## Impact

- New: `tgscout/` workspace, `e2e/`, `scripts/qa.sh`, `.go-arch-lint.yml`, `.golangci.yml`, `_typos.toml`, `.jscpd.json`, `metrics/cyclomatic.tsv`, `scripts/check-complexity-ratchet.sh`, `.githooks/pre-commit`, `.env.example`, `tgscout.yaml.example`, `README.md` run section, the first migration under `adapters/secondary/persistence/sqlite/migrations/`.
- `AGENTS.md`: § Gherkin and § How to verify a change get the real runner and commands.
- CI needs cgo for `mattn/go-sqlite3`.

## Open points

- Command shape: plain `tgscout` or `tgscout run`. Proposed: plain `tgscout` with flags (`--config`, `--db`), since version 1 has one mode.
- Default place of `tgscout.db`: next to the config file. Proposed: next to the config file, overridable by a flag.
- Migrations: an embedded SQL file with a tiny runner, or `golang-migrate`. Proposed: embedded SQL files (`embed`) with a version table; check the kit tech matrix at design.
- Gherkin runner: proposed `godog`, with `.feature` files under `e2e/features/` and slice tests next to their component; settle at design with `testing-*-go.md`.
- Gap in the design board: the plan's review step (`proceed-design review`) was not run before filing; any mismatch it finds goes into the slice it touches.

## Release

SemVer: MINOR — the first working command of tgscout.
Target version: 0.1.0.
