@cli_lifecycle @e2e
Feature: cli-lifecycle — how the tgscout command starts, opens its stores and stops

  Rule: Config file at start

    Scenario: the config file is missing
      Given no config file at the path the owner gives
      When the owner starts tgscout
      Then tgscout prints an error that names the missing path
      And it exits with a non-zero code
      And no database file is created

    Scenario: the config file is not valid YAML
      Given a config file with broken YAML on its fourth line
      When the owner starts tgscout
      Then tgscout prints an error that names line 4
      And it exits with a non-zero code

  Rule: Database at start

    Scenario: a first start creates the database
      Given a valid config file and no database file yet
      When the owner starts tgscout
      Then the database file exists next to the config file
      And it holds the chat cursor, processed message and decision tables
      And the console says the database was created

    Scenario: a second start keeps the database
      Given a database created by an earlier start, with rows in it
      When the owner starts tgscout again
      Then the rows are still there
      And the schema is unchanged
      And the console says the schema is up to date

  Rule: Clean stop

    Scenario: tgscout starts and stops cleanly
      Given a valid config file and no database file yet
      When the owner starts tgscout and then presses Ctrl+C
      Then tgscout says that it stopped
      And it exits with code 0
      And the database holds the chat cursor, processed message and decision tables

    Scenario: tgscout stops on a terminate signal
      Given tgscout is running with a valid config file
      When the system asks tgscout to terminate
      Then tgscout says that it stopped
      And it exits with code 0

  Rule: Command usage

    Scenario: the owner asks for help
      Given no config file at all
      When the owner asks tgscout for help
      Then tgscout prints the usage with every option and its default
      And it exits with code 0

    Scenario: the owner gives an unknown option
      Given a valid config file
      When the owner starts tgscout with an option it does not know
      Then tgscout prints the error and the usage
      And it exits with code 2
      And no database file is created
