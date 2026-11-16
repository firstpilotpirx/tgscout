@settings @e2e
Feature: settings — what the owner writes into the config file and how tgscout checks it

  Rule: Config file fields

    Scenario: every form of chat reference is accepted
      Given a config with a request whose chats are a username, a public chat link and a numeric ID
      When the owner starts tgscout
      Then tgscout prints that the config has 1 request, 3 chats and the model example-model

  Rule: Catch-up depth override
    # The resolution scenarios of this requirement run as unit tests of the
    # config parser (design § Test levels): no console output shows a depth yet.

  Rule: Config validation

    Scenario Outline: an invalid config stops the start
      Given a config with <problem>
      When the owner starts tgscout
      Then tgscout prints an error naming <field> and its line
      And it exits with a non-zero code
      And no database file is created

      Examples:
        | problem                  | field             |
        | no requests              | requests          |
        | a request without chats  | requests[0].chats |
        | two requests with one id | requests[1].id    |
        | an unknown key           | colour            |
        | a zero depth             | catch_up.days     |
        | an api_hash key          | api_hash          |

    Scenario: every problem is listed at once
      Given a config with a request without chats, a second request with the same id and an api_hash key
      When the owner starts tgscout
      Then tgscout says the config has 3 errors
      And it prints one line for each of the three problems, each with its line

  Rule: Chat count warning

    Scenario: too many chats is a warning
      Given a config with 101 distinct chats
      When the owner starts tgscout
      Then tgscout warns that the design maximum is 100 chats
      And it goes on with the start

  Rule: Start summary

    Scenario: a valid config prints the summary
      Given a config with 2 requests over 3 distinct chats and a model named example-model
      When the owner starts tgscout
      Then tgscout prints that the config has 2 requests, 3 chats and the model example-model
