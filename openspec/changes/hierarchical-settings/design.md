# Design

## Context

`walking-skeleton` left a method-less `SettingsRepository` port in `core/application`, an empty `core/domain`, and `yaml.CheckConfigFile` in `adapters/secondary/settings/yaml`, which checks with viper only that the file exists and is valid YAML. `apps/cli/start.go` prints `✓ config <path>` and then opens the database. The minimal "valid config" of the existing tests (one request, no model, no owner) becomes invalid under this change, so those fixtures grow.

## Goals / Non-Goals

**Goals:**

- One resolved, immutable view of `tgscout.yaml` for the core: owner, workers, LM Studio, and every request with its chats and the effective catch-up depth per (request, chat) pair.
- Every config problem reported at once with its line and field; the start stops before the database.
- The warning for more than 100 chats and the one-line summary of the agreed mock.

**Non-Goals:**

- Overrides other than `catch_up`; reloading the file while running; checking chats in Telegram; deciding which of `messages` and `days` wins (the `catch-up` change).

## Decisions

### Domain types (`core/domain`)

- `ChatRef` — a string value type; `NewChatRef(text)` accepts `@username` (5–32 letters, digits, `_`, starting with a letter), `https://t.me/<username>` / `t.me/<username>`, or a numeric ID (optional `-`, digits). `Key()` gives the normalized form used to count distinct chats: a link and `@name` of the same username are one chat, case-insensitive.
- `Owner` — same forms as a chat reference except the link: `@username` or a numeric ID.
- `CatchUp{Messages, Days int}` — the resolved depth.
- `RequestChat{Ref ChatRef, CatchUp CatchUp}`, `Request{ID, Find string, Chats []RequestChat}`.
- `LmStudio{URL string, Model string, Timeout time.Duration}`.
- `Settings{Owner, Workers int, LmStudio, Requests []Request}` with `DistinctChats() int`.

Resolution lives in the adapter: the domain receives only resolved values, so later slices never see the hierarchy.

### Port (`core/application`)

`SettingsRepository` gets one method: `Settings() domain.Settings`. The file is read once, at start, by the adapter's constructor; the port only returns the resolved view.

### Adapter (`adapters/secondary/settings/yaml`)

- `Parse(data []byte) (domain.Settings, []string, error)` — pure, used by unit tests: parses with `github.com/goccy/go-yaml` (`parser.ParseBytes`) into its AST and walks the nodes by hand. Walking the AST (instead of `yaml.Strict()` decoding, which stops at the first problem) lets it collect every error with the line of its node. The second result is the warnings.
- `LoadSettings(path) (*YamlSettingsRepository, error)` — reads the file and calls `Parse`. Errors: `ErrConfigNotFound` (missing file), `*ConfigSyntaxError` (not YAML, with its line), `*ConfigErrors{Path, Errors []FieldError}` with `FieldError{Line int, Field, Reason string}`; `Warnings()` returns the warnings. `CheckConfigFile` and viper go away; viper leaves `go.mod` and `.go-arch-lint.yml`, `goccy` (`go-yaml` v1.19.2, from 2026-01) joins the adapters' vendors.
- Field paths are written as `requests[1].chats[0].catch_up.days`; a missing key is reported on the line of the mapping that should hold it (root: line 1).
- Secret keys (`api_id`, `api_hash`, `session`) are reported wherever they appear and are not reported again as unknown keys.
- Defaults: `workers` 2, `lm_studio.url` `http://localhost:1234/v1`, `lm_studio.timeout` 60s, `catch_up` 500 messages and 3 days. Resolution per field: chat → request → root → default.
- Errors are sorted by line, then by field.

### Console (`apps/cli`)

Follows the agreed mock: the config line becomes `✓ <path>: <n> requests, <m> chats, model <model>`; an invalid config prints `✗ <path>: <n> errors` (`1 error` for one) and one indented line per error `line <n>  <field>  <reason>` with the line number and the field padded to columns; the warning is `⚠ <path>: <m> chats; the design maximum is 100` (plain mark `[warn]`). Errors and the warning go to standard error; the summary to standard output. The `console` gets a `warn` method and a `warn` mark.

### Files

- `tgscout.yaml.example` — every field with made-up values and a placeholder owner (`@your_username`).
- `.gitignore` — `tgscout.yaml` (the owner's real config).
- `README.md` — config section.
- `docs/architecture/overview.html` § Config — the sample uses the L3 names (`catch_up`, `lm_studio`, `id`, `find`, `ref`).

### Test levels

| Scenario | Level |
|---|---|
| defaults fill what the file leaves out | unit (`Parse`) |
| every form of chat reference is accepted | unit (`Parse`, `NewChatRef`) |
| the root depth applies when nothing overrides it | unit (`Parse`) |
| a request overrides the root | unit (`Parse`) |
| a chat overrides its request | unit (`Parse`) |
| the same chat in two requests keeps two depths | unit (`Parse`) |
| built-in defaults when the root sets no depth | unit (`Parse`) |
| an invalid config stops the start (outline) | E2E (`e2e/features/settings.feature`); each problem also a unit case of `Parse` |
| every problem is listed at once | E2E |
| too many chats is a warning | E2E |
| a valid config prints the summary | E2E |
| the config file is missing (cli-lifecycle) | E2E (existing, unchanged) |
| the config file is not valid YAML (cli-lifecycle) | E2E (existing, unchanged) |

The adapter's file reading (`LoadSettings`) gets an integration test (`-tags=integration`), as `CheckConfigFile` had.

## Interface mock

[mock/index.html](mock/index.html) — the agreed console output: good start, invalid config, too many chats, in a dark and a light terminal.

## Risks / Trade-offs

- Walking the AST by hand is more code than decoding into structs → kept in small per-level functions (root, request, chat, catch-up) to stay under the lint limits.
- `NewChatRef` rules may be stricter or looser than Telegram's → `telegram-sign-in` resolves the reference and reports a chat that does not exist.
