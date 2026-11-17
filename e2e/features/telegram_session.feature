@telegram_session @e2e
Feature: telegram-session — how tgscout signs in to its Telegram account and finds its chats

  Rule: Telegram credentials

    Scenario: the api hash is missing
      Given a valid config file
      And an api id but no api hash
      When the owner starts tgscout
      Then tgscout prints an error that names the missing api hash
      And it exits with a non-zero code

  Rule: First sign-in

    Scenario: first start signs in and saves the session
      Given a valid config file
      And no session file yet and an account with a two-step password
      When the owner starts tgscout on a terminal and types the phone number, the login code and the password
      Then tgscout says it signed in as the account's name
      And the session file exists and only its owner can read or write it
      And the password is not shown while typed

    Scenario: a wrong login code
      Given a valid config file
      And no session file yet and an account with a two-step password
      When the owner starts tgscout on a terminal and types a wrong login code
      Then tgscout says the code is wrong and asks for the code again
      And no session file is saved

  Rule: First sign-in needs a terminal

    Scenario: first sign-in without a terminal
      Given a valid config file
      And no session file yet and an account with a two-step password
      When the owner starts tgscout with its input not coming from a terminal
      Then tgscout prints an error that says to sign in once from a terminal
      And it exits with a non-zero code

  Rule: Later sign-in from the saved session

    Scenario: a later start signs in silently
      Given a valid config file
      And a valid private session file
      When the owner starts tgscout
      Then tgscout says it signed in from the saved session without asking anything
      And it goes on running

    Scenario: the session file is open to others
      Given a valid config file
      And a session file readable by other users
      When the owner starts tgscout
      Then tgscout prints an error that says how to make the session file private
      And it exits with a non-zero code

    Scenario: a revoked session
      Given a valid config file
      And a private session file that Telegram no longer accepts
      When the owner starts tgscout
      Then tgscout prints an error that tells the owner to sign in again
      And it exits with a non-zero code

  Rule: Watched chats lookup

    Scenario: every chat and the owner are found
      Given a config whose 2 chats the account has joined and whose owner Telegram knows
      When the owner starts tgscout
      Then tgscout says it found the owner and watches 2 chats
      And it goes on running

    Scenario: a chat the account has not joined
      Given a config where one chat the account has not joined and one chat does not exist
      When the owner starts tgscout
      Then tgscout prints an error that lists both chats, each with its reason
      And it exits with a non-zero code

    Scenario: Telegram asks to wait during the lookup
      Given a valid config file
      And Telegram asks to wait 1 second before the lookup
      When the owner starts tgscout
      Then tgscout prints that it waits 1s
      And it watches 1 chat

  Rule: Owner lookup

    Scenario: the owner is not found
      Given a config whose owner is a numeric ID that is not among the account's contacts
      When the owner starts tgscout
      Then tgscout prints an error that says to add the owner to the account's contacts
      And it exits with a non-zero code

    Scenario: the owner never wrote to tgscout's account
      Given a config whose owner Telegram knows but who never wrote to tgscout's account
      When the owner starts tgscout
      Then tgscout stops with an error that it cannot write to the owner and a hint to write «hi» to its account once
      And it exits with a non-zero code

  Rule: Secrets stay out of the output

    Scenario: secrets never reach the console
      Given a valid config file
      And no session file yet and an account with a two-step password
      When the owner starts tgscout on a terminal and types a wrong code, then the right code and the password
      Then the output holds no phone number, api hash, login code, password or session data
