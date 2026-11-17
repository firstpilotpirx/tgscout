## MODIFIED Requirements

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
