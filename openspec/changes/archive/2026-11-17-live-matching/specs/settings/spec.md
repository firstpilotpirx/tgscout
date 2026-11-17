# Spec Delta

## MODIFIED Requirements

### Requirement: Config file fields

The config file SHALL hold at the root `owner` (required), `workers` (default 2), `queue_size` (default 1000), `lm_studio` (`url`, `model` required, `timeout`), `catch_up` (`messages`, `days`) and one or more `requests`. A request SHALL hold a unique `id`, the `find` text, one or more `chats` and an optional `catch_up`. A chat SHALL be a chat reference (`@username`, a public `t.me` link or a numeric ID) or an object with `ref` and an optional `catch_up`.

#### Scenario: defaults fill what the file leaves out

- **GIVEN** a config with an owner, one request with one chat and an LM Studio model, and nothing else
- **WHEN** tgscout loads the config
- **THEN** it uses 2 workers and a queue of 1000 messages
- **AND** it calls LM Studio at the local default address with a 60 second timeout

#### Scenario: every form of chat reference is accepted

- **GIVEN** a request whose chats are a username, a public chat link and a numeric ID of a private chat
- **WHEN** tgscout loads the config
- **THEN** the request watches those three chats

### Requirement: Config validation

tgscout SHALL check the whole config before it opens the database, list every problem at once, one line each with its line and field, and exit with code 1. A problem is: no requests; a request without `id`, `find` or chats; a duplicate `id`; an unknown key; a wrong kind of value; a depth below 1; an `lm_studio.url` that is not http or https; no `lm_studio.model`; `workers` below 1; `queue_size` below 1; a missing or malformed owner or chat reference; a secret key (`api_id`, `api_hash`, `session`) anywhere.

#### Scenario Outline: an invalid config stops the start

- **GIVEN** a config with <problem>
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints an error naming <field> and its line
- **AND** it exits with a non-zero code
- **AND** no database file is created

Examples: no requests / `requests`; a request without chats / `chats`; two requests with one id / `id`; an unknown key / that key; a zero queue size / `queue_size`; a zero depth / `catch_up`; an `api_hash` key / `api_hash`.

#### Scenario: every problem is listed at once

- **GIVEN** a config with a request without chats, a second request with the same id and an `api_hash` key
- **WHEN** the owner starts tgscout
- **THEN** tgscout says the config has 3 errors
- **AND** it prints one line for each of the three problems, each with its line
