# telegram-session Specification

## Purpose
How tgscout signs in to its own Telegram account at start: the credentials it reads, the first interactive sign-in, the private session file that later starts reuse, the lookup of the watched chats and the owner, and the secrets that never reach the console.

## Requirements

### Requirement: Telegram credentials

tgscout SHALL read the Telegram `api_id`, `api_hash` and the session file path from the environment, filled from a `.env` file in the current folder when it exists; a variable already set in the environment SHALL win over the file. A missing `api_id` or `api_hash` SHALL stop the start, after the config and the database, with an error that names the missing variable and says to fill it in `.env`. Without a session file path tgscout SHALL keep the session in a file next to the config file.

#### Scenario: the api hash is missing

- **GIVEN** a valid config file and an api id but no api hash
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that names the missing api hash
- **AND** it exits with a non-zero code

### Requirement: First sign-in

When no session file exists, tgscout SHALL ask in the console for the phone number, then the login code Telegram sends, then the two-step password when the account has one; the password SHALL be hidden while typed. A wrong login code or password SHALL be reported and asked again, and no session file SHALL be saved before the sign-in succeeds. After the sign-in succeeds, tgscout SHALL save the session file readable and writable by the owner only and print where it saved it.

#### Scenario: first start signs in and saves the session

- **GIVEN** no session file yet and an account with a two-step password
- **WHEN** the owner starts tgscout and types the phone number, the login code and the password
- **THEN** tgscout says it signed in as the account's name
- **AND** the session file exists and only its owner can read or write it
- **AND** the password is not shown while typed

#### Scenario: a wrong login code

- **GIVEN** no session file yet
- **WHEN** the owner types a wrong login code
- **THEN** tgscout says the code is wrong and asks for the code again
- **AND** no session file is saved

### Requirement: First sign-in needs a terminal

When no session file exists and standard input is not a terminal, tgscout SHALL stop with an error that says to sign in once from a terminal, and ask nothing.

#### Scenario: first sign-in without a terminal

- **GIVEN** no session file yet
- **WHEN** the owner starts tgscout with its input not coming from a terminal
- **THEN** tgscout stops with an error that says to sign in once from a terminal
- **AND** it exits with a non-zero code

### Requirement: Later sign-in from the saved session

When the session file exists, tgscout SHALL first check its permissions: a file that users other than its owner may read or write SHALL stop the start with an error that names the file, its mode and the command that makes it private. A private session file SHALL sign tgscout in without asking anything. A session Telegram no longer accepts SHALL stop the start with an error that tells the owner to delete the session file and start tgscout from a terminal to sign in again.

#### Scenario: a later start signs in silently

- **GIVEN** a valid private session file
- **WHEN** the owner starts tgscout
- **THEN** tgscout says it signed in from the saved session without asking anything
- **AND** it goes on running

#### Scenario: the session file is open to others

- **GIVEN** a session file readable by other users
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that says how to make the file private
- **AND** it exits with a non-zero code

#### Scenario: a revoked session

- **GIVEN** a private session file that Telegram no longer accepts
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops and tells the owner to sign in again
- **AND** it exits with a non-zero code

### Requirement: Watched chats lookup

After sign-in tgscout SHALL look up every distinct chat of the config. A chat that does not exist or that the account has not joined SHALL stop the start with one error that lists every such chat with its reason. When Telegram asks to wait during the lookup, tgscout SHALL print how long it waits, wait that time and go on. On success tgscout SHALL print how many chats it watches, then run.

#### Scenario: every chat and the owner are found

- **GIVEN** a config whose chats the account has all joined and whose owner Telegram knows
- **WHEN** the owner starts tgscout
- **THEN** tgscout says it found the owner and watches every chat of the config
- **AND** it goes on running

#### Scenario: a chat the account has not joined

- **GIVEN** a config with one chat the account has not joined and one chat that does not exist
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that lists both chats, each with its reason
- **AND** it exits with a non-zero code

#### Scenario: Telegram asks to wait during the lookup

- **GIVEN** a config whose chats Telegram looks up only after a short wait
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints how long it waits
- **AND** after the wait it watches every chat of the config

### Requirement: Owner lookup

After sign-in tgscout SHALL find the owner: an owner given as `@username` through Telegram, an owner given as a numeric ID among the account's contacts. An owner not found SHALL stop the start with an error that says to add the owner to the account's contacts; a found owner SHALL be printed as written in the config.

#### Scenario: the owner is not found

- **GIVEN** a config whose owner is a numeric ID that is not among the account's contacts
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that says to add the owner to the account's contacts
- **AND** it exits with a non-zero code

### Requirement: Secrets stay out of the output

tgscout SHALL never print or log the phone number, the `api_hash`, the login code, the two-step password or any session data, in a successful start or a failed one; it SHALL name the signed-in account by its name only.

#### Scenario: secrets never reach the console

- **GIVEN** a first start where the owner types a wrong code, then the right code and the password
- **WHEN** tgscout prints its output
- **THEN** the output holds no phone number, api hash, login code, password or session data
