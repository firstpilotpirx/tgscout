## MODIFIED Requirements

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

#### Scenario: a negative history pause is an error

- **GIVEN** a config with `history_pause: -1s`
- **WHEN** tgscout loads the config
- **THEN** it stops with an error on the field `history_pause`
