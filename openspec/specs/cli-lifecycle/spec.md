# cli-lifecycle Specification

## Purpose
How the `tgscout` command starts: it checks the config file, opens and migrates its SQLite database, runs until a stop signal, stops cleanly, and prints its usage on request or on a wrong option.

## Requirements

### Requirement: Config file at start

`tgscout` SHALL read its config file (`--config`, default `./tgscout.yaml`) before anything else and report a valid file with its path and its summary (the `settings` capability). A missing file SHALL give an error naming the path and a hint to copy the example, exit code 1 and no database. A file that is not valid YAML SHALL give an error naming the path and the line of the problem, and exit code 1. A file with invalid fields SHALL give the errors the `settings` capability lists, exit code 1 and no database.

#### Scenario: the config file is missing

- **GIVEN** no config file at the path the owner gives
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints an error that names the missing path
- **AND** it exits with a non-zero code
- **AND** no database file is created

#### Scenario: the config file is not valid YAML

- **GIVEN** a config file with broken YAML on its fourth line
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints an error that names line 4
- **AND** it exits with a non-zero code

### Requirement: Database at start

After the config, `tgscout` SHALL open its SQLite file (`--db`, default `tgscout.db` beside the config file). A new file SHALL get the schema: chat cursor, processed messages, decisions. An existing file SHALL get only the missing schema steps and keep every row. The console SHALL say "created" or "schema up to date". A file that cannot be opened or migrated SHALL give an error naming the path, and exit code 1.

#### Scenario: a first start creates the database

- **GIVEN** a valid config file and no database file yet
- **WHEN** the owner starts tgscout
- **THEN** the database file exists next to the config file
- **AND** it holds the chat cursor, processed message and decision tables
- **AND** the console says the database was created

#### Scenario: a second start keeps the database

- **GIVEN** a database created by an earlier start, with rows in it
- **WHEN** the owner starts tgscout again
- **THEN** the rows are still there
- **AND** the schema is unchanged
- **AND** the console says the schema is up to date

### Requirement: Clean stop

While it runs, `tgscout` SHALL say on the console that it is running and how to stop it. On an interrupt (Ctrl+C, SIGINT) or a terminate signal (SIGTERM), it SHALL stop as the delivery capability's graceful stop says, close the database, say that it stopped and exit with code 0.

#### Scenario: tgscout starts and stops cleanly

- **GIVEN** a valid config file and no database file yet
- **WHEN** the owner starts tgscout and then presses Ctrl+C
- **THEN** tgscout says that it stopped
- **AND** it exits with code 0
- **AND** the database holds the chat cursor, processed message and decision tables

#### Scenario: tgscout stops on a terminate signal

- **GIVEN** tgscout is running with a valid config file
- **WHEN** the system asks tgscout to terminate
- **THEN** tgscout says that it stopped
- **AND** it exits with code 0

### Requirement: Command usage

`tgscout` SHALL be one command with the flags `--config`, `--db` and `-h`/`--help`, no subcommands. `--help` SHALL print a one-line description, the usage and every flag with its default to standard output, exit code 0. An unknown flag or a stray argument SHALL print the error and the usage to standard error, exit code 2. Without a terminal or with `NO_COLOR` set, output SHALL have no color and `[ok]` / `[fail]` replace `✓` / `✗`.

#### Scenario: the owner asks for help

- **GIVEN** no config file at all
- **WHEN** the owner asks tgscout for help
- **THEN** tgscout prints the usage with every option and its default
- **AND** it exits with code 0

#### Scenario: the owner gives an unknown option

- **GIVEN** a valid config file
- **WHEN** the owner starts tgscout with an option it does not know
- **THEN** tgscout prints the error and the usage
- **AND** it exits with code 2
- **AND** no database file is created
