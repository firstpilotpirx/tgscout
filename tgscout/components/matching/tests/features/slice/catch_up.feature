@catch_up @slice
Feature: catch-up — the history collector through the real use cases

  Rule: Where catch-up starts and stops

    Scenario: the message limit comes first
      Given a watched chat with no cursor and 700 messages posted in the last day
      When the owner starts tgscout
      Then tgscout judges only the newest 500 of them

  Rule: Gap after a long pause

    Scenario: a cursor older than the depth
      Given a chat whose cursor is at a message of 12 days ago and newer messages over all those days
      When the owner starts tgscout
      Then tgscout judges only the messages of the last 3 days
      And the console line of that chat says it has a gap

  Rule: Depth per request

    Scenario: a request overrides the depth
      Given a request with a catch-up depth of 7 days
      When catch-up reads a chat of that request for the first time
      Then it reads back 7 days for that request

    Scenario: a chat overrides the depth
      Given a chat of a request with a catch-up depth of 2,000 messages
      When catch-up reads that chat for the first time
      Then it reads up to 2,000 messages

    Scenario: two requests with different depths on one chat
      Given a chat bound to a request with 7 days and a request with 3 days, and a message of 5 days ago
      When catch-up reads that chat for the first time
      Then the message is judged against the 7-day request only

  Rule: Nothing judged twice

    Scenario: a processed message is not judged again
      Given a message after the cursor that is already marked processed
      When catch-up reaches it
      Then it is skipped and no second match is sent

  Rule: Gap read after a reconnect

    Scenario: a message processed during the drop is not judged twice
      Given a message reached tgscout and was processed just before the connection dropped
      When the connection is back and tgscout reads the gap
      Then that message is not judged again
