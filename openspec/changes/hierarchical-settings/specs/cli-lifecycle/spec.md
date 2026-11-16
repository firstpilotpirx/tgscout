# Spec Delta

## MODIFIED Requirements

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
