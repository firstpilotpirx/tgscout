## Why

The owner tells tgscout what to look for only through `tgscout.yaml`: many standing requests, each bound to its own set of chats (one or more), plus the catch-up depth and the LM Studio settings. The design makes settings hierarchical — set once at the root, overridden for one request, overridden again for one chat of that request; the nearest level wins (chat, then request, then root). Every later slice reads its values from here, so the schema, the defaults, the override rules and the error messages must be right and tested before any message is read.

Design pages this slice implements:

- [L2 Container § Config](../../../docs/architecture/overview.html) — hierarchical settings and the sample file; secrets never in this file.
- [L3 Component — tgscout § Data structures](../../../docs/architecture/tgscout.html) — the `tgscout.yaml` entity picture: root `workers`, `lm_studio {url, model, timeout}`, `catch_up {messages = 500, days = 3}`, `request {id, find}`, `chat {ref}`, overrides on request and chat.
- [L4 Code — matching](../../../docs/modules/matching/overview.html) — port `SettingsRepository`, adapter `YamlSettingsRepository` (`secondary/settings/yaml`).
- [Start flow](../../../docs/flows/start.html) — «config valid? → no: print the error and exit».

## What Changes

- `YamlSettingsRepository` reads `tgscout.yaml` once at start and gives the core one resolved view through the `SettingsRepository` port: every request with its text and its chats, and for every (request, chat) pair the effective catch-up depth.
- Field names follow the L3 entity picture (`catch_up`, `lm_studio`, `id`, `find`); the L2 sample on the board is updated to them in this change.
- Root settings: `workers` (number of matcher workers, default 2), `lm_studio` (`url` default `http://localhost:1234/v1`, `model` required, `timeout` default 60s), `catch_up` (`messages`, `days`; defaults 500 and 3), `owner` (`@username` or numeric ID) to write to.
- Request settings: `id`, `find` (the text the LLM compares with), `chats` (one or more), optional `catch_up` override.
- Chat settings: a plain chat reference (`@username`, a public `t.me` link, or a numeric ID for a private chat), or an object with the reference and an optional `catch_up` override. Only `catch_up` can be overridden on a request or a chat.
- Resolution rule: chat value, else request value, else root value, else the built-in default; each field resolves on its own (a chat may override only `messages` and keep the request's `days`).
- The file is parsed with `github.com/goccy/go-yaml` (line numbers, strict mode for unknown keys).
- Validation lists every error at once, one line each, then stops the start with exit code 1; each error names the field and its line: no requests; a request without `find` or without chats; two requests with the same `id`; an unknown key; a negative or zero depth; a bad `lm_studio.url`; `workers` below 1; a secret-looking key (`api_id`, `api_hash`, `session`) in the file.
- More than 100 distinct chats (the design maximum) gives a warning, not an error.
- On a good start tgscout prints one line: how many requests, how many chats, which model.
- `tgscout.yaml.example` shows every field with made-up values; `tgscout.yaml` is added to `.gitignore`.
- Console output follows the agreed mock: [mock/index.html](mock/index.html).

Current stage: PoC. Target stage: PoC.
Scope: the schema of `tgscout.yaml`, defaults, the three-level override rule (for `catch_up` only), validation and its errors, the start summary line, the example file, `tgscout.yaml` in `.gitignore`, the L2 sample field names.
Acceptance bar: the scenarios below pass as unit tests of the resolution and slice tests of the start; `tgscout` with the example file prints the summary line.
Out of scope: overriding any setting other than `catch_up` per request or chat (for example the LLM model); reloading the file while tgscout runs (read at start only); checking that chats exist in Telegram (`telegram-sign-in`); using the values (later slices); secrets (they stay in `.env` and the session file).
Interface: CLI — `tgscout.yaml` format, start errors and the summary line on the console.

## Acceptance scenarios

- Scenario: the root depth applies when nothing overrides it
  - Given a request whose chat sets no catch-up depth and a root depth of 500 messages and 3 days
  - When tgscout loads the config
  - Then that chat of that request has 500 messages and 3 days
- Scenario: a request overrides the root
  - Given a request with a catch-up depth of 7 days
  - When tgscout loads the config
  - Then every chat of that request has 7 days and the root number of messages
- Scenario: a chat overrides its request
  - Given a request with 7 days and one of its chats with 2,000 messages
  - When tgscout loads the config
  - Then that chat has 2,000 messages and 7 days, and the other chats of the request keep the request's depth
- Scenario: the same chat in two requests keeps two depths
  - Given a chat bound to two requests with different depths
  - When tgscout loads the config
  - Then each request sees its own depth for that chat
- Scenario: built-in defaults when the root sets no depth
  - Given a config without a root catch-up depth
  - When tgscout loads the config
  - Then every chat has 500 messages and 3 days
- Scenario Outline: an invalid config stops the start
  - Given a config with <problem>
  - When the owner starts tgscout
  - Then tgscout prints an error naming <field> and its line and exits with a non-zero code
  - Examples: no requests / `requests`; a request without chats / `chats`; two requests with one id / `id`; an unknown key / that key; a zero depth / `catch_up`; an `api_hash` key / `api_hash`
- Scenario: too many chats is a warning
  - Given a config with 101 distinct chats
  - When the owner starts tgscout
  - Then tgscout warns that the design maximum is 100 chats and goes on

## Capabilities

### New Capabilities

- `settings`: the `tgscout.yaml` schema, defaults, the chat → request → root override rule and its validation.

## Impact

- New: `components/matching/adapters/secondary/settings/yaml/`, domain types `Request` and the resolved settings in `core/domain`, the `SettingsRepository` port in `core/application`.
- Changed: `tgscout.yaml.example`, `.gitignore`, `README.md` (config section), `apps/cli/main.go` (wiring and the summary line), `docs/architecture/overview.html` (L2 sample field names).

## Decided points

- Field names: the L3 names (`catch_up`, `lm_studio`, `id`, `find`); the L2 sample on the board is updated in this change.
- Chat reference: `@username` or a public link, a numeric ID for a private chat; the lookup itself is in `telegram-sign-in`.
- Owner: `owner` in `tgscout.yaml` as `@username` or numeric ID; the example holds a placeholder.
- YAML library: `github.com/goccy/go-yaml` (the owner's answer), replacing viper (the tech-matrix "Config / env" pick and the walking-skeleton choice) for line numbers and every error at once; a later slice that reads `.env` picks its own reader.
- When both `messages` and `days` are set, which limit wins is decided in `catch-up`; this slice only stores both.

## Release

SemVer: MINOR — new config format the owner writes.
Target version: 0.2.0.
