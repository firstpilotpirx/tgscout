# settings Specification

## Purpose
What the owner writes into the config file: the requests to look for, the chats each request watches, the owner to write to, the LM Studio settings and the catch-up depth, which a request and a chat of that request may override.

## Requirements

### Requirement: Config file fields

The config file SHALL hold at the root `owner` (required), `workers` (default 2), `queue_size` (default 1000), `retry_max_pause` (default 60 seconds), `stop_grace` (default 10 seconds), `status_interval` (default 5 minutes), `history_pause` (default 1 second), `lm_studio` (`url`, `model` required, `timeout`), `catch_up` (`messages`, `days`) and one or more `requests`. `retry_max_pause` and `stop_grace` SHALL be positive durations; `status_interval` SHALL be a duration of at least 10 seconds; `history_pause` SHALL be a duration of 0 seconds or more, where 0 turns the pause off. A request SHALL hold a unique `id`, the `find` text, one or more `chats` and an optional `catch_up`. A chat SHALL be a chat reference (`@username`, a public `t.me` link or a numeric ID) or an object with `ref` and an optional `catch_up`.

#### Scenario: defaults fill what the file leaves out

- **GIVEN** a config with an owner, one request with one chat and an LM Studio model, and nothing else
- **WHEN** tgscout loads the config
- **THEN** it uses 2 workers and a queue of 1000 messages
- **AND** it calls LM Studio at the local default address with a 60 second timeout
- **AND** it pauses at most 60 seconds between retries and gives messages in work 10 seconds at a stop
- **AND** it prints a status line every 5 minutes
- **AND** it keeps 1 second between two history calls

#### Scenario: every form of chat reference is accepted

- **GIVEN** a request whose chats are a username, a public chat link and a numeric ID of a private chat
- **WHEN** tgscout loads the config
- **THEN** the request watches those three chats

### Requirement: Catch-up depth override

For every pair of a request and one of its chats, tgscout SHALL resolve the catch-up depth field by field: the chat's value, else the request's value, else the root value, else the built-in default (500 messages, 3 days). The same chat bound to two requests SHALL keep a depth per request. Only `catch_up` MAY be set on a request or a chat; tgscout SHALL read the file once, at start.

#### Scenario: the root depth applies when nothing overrides it

- **GIVEN** a request whose chat sets no catch-up depth and a root depth of 500 messages and 3 days
- **WHEN** tgscout loads the config
- **THEN** that chat of that request has 500 messages and 3 days

#### Scenario: a request overrides the root

- **GIVEN** a root depth of 500 messages and 3 days and a request with a catch-up depth of 7 days
- **WHEN** tgscout loads the config
- **THEN** every chat of that request has 500 messages and 7 days

#### Scenario: a chat overrides its request

- **GIVEN** a request with 7 days and one of its chats with 2,000 messages
- **WHEN** tgscout loads the config
- **THEN** that chat has 2,000 messages and 7 days
- **AND** the other chats of the request keep the request's depth

#### Scenario: the same chat in two requests keeps two depths

- **GIVEN** a chat bound to two requests with different depths
- **WHEN** tgscout loads the config
- **THEN** each request sees its own depth for that chat

#### Scenario: built-in defaults when the root sets no depth

- **GIVEN** a config without a root catch-up depth
- **WHEN** tgscout loads the config
- **THEN** every chat has 500 messages and 3 days

### Requirement: Config validation

tgscout SHALL check the whole config before it opens the database, list every problem at once, one line each with its line and field, and exit with code 1. A problem is: no requests; a request without `id`, `find` or chats; a duplicate `id`; an unknown key; a wrong kind of value; a depth below 1; an `lm_studio.url` that is not http or https; no `lm_studio.model`; `workers` below 1; `queue_size` below 1; a `retry_max_pause` or `stop_grace` that is not a positive duration; a `status_interval` that is not a duration or is below 10 seconds; a `history_pause` that is not a duration or is below 0 seconds; a missing or malformed owner or chat reference; a secret key (`api_id`, `api_hash`, `session`) anywhere.

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

#### Scenario: a retry cap that is not a duration is refused

- **GIVEN** a config whose retry cap is not a positive duration
- **WHEN** tgscout loads the config
- **THEN** it names the field and the line, and does not start

#### Scenario: a status interval below 10 seconds is refused

- **GIVEN** a config whose status interval is 5 seconds
- **WHEN** tgscout loads the config
- **THEN** it names the status interval field and its line, and says it must be at least 10 seconds
- **AND** it does not start

#### Scenario: a negative history pause is an error

- **GIVEN** a config with `history_pause: -1s`
- **WHEN** tgscout loads the config
- **THEN** it stops with an error on the field `history_pause`

### Requirement: Chat count warning

When the config names more than 100 distinct chats (the design maximum), tgscout SHALL print a warning with the count and the maximum and go on with the start.

#### Scenario: too many chats is a warning

- **GIVEN** a config with 101 distinct chats
- **WHEN** the owner starts tgscout
- **THEN** tgscout warns that the design maximum is 100 chats
- **AND** it goes on with the start

### Requirement: Start summary

On a valid config, tgscout SHALL print one line with the config path, the number of requests, the number of distinct chats and the LM Studio model.

#### Scenario: a valid config prints the summary

- **GIVEN** a config with 2 requests over 3 distinct chats and a model named example-model
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints that the config has 2 requests, 3 chats and the model example-model
