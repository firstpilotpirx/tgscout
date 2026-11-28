@catch_up @e2e
Feature: catch-up — at start tgscout reads what the watched chats posted while it was off

  Rule: Catch-up at start

    Scenario: a match in the history reaches the owner
      Given a matching message posted while tgscout was off
      When the owner starts tgscout and it catches up
      Then the owner gets the history match as a private message

  Rule: Where catch-up starts and stops

    Scenario: first start reads the default depth
      Given a watched chat with no cursor and 4 days of history
      When the owner starts tgscout and it catches up
      Then tgscout judges the messages of the last 3 days only

    Scenario: a restart reads only what it missed
      Given a chat whose cursor is at message 100 and 20 newer messages
      When the owner starts tgscout and it catches up
      Then only the 20 newer messages are judged

    # The message limit runs in the slice feature (design § Test levels).

  Rule: Gap after a long pause
    # A cursor older than the depth runs in the slice feature.

  Rule: Depth per request
    # The request and chat overrides and two depths on one chat run in the slice feature.

  Rule: Nothing judged twice
    # A processed message met by catch-up runs in the slice feature.

    Scenario: live and catch-up at once
      Given catch-up is reading a chat
      When a new message appears in that chat while it is read
      Then that message is judged once and the owner gets one match for it

  Rule: Cursor during catch-up
    # The cursor hold runs as a unit test of the cursor and the catch-up use case.

  Rule: Telegram asks to wait during a history read

    Scenario: Telegram asks to wait during catch-up
      Given Telegram asks to wait 1 second on the first history read
      When the owner starts tgscout and it catches up
      Then the console shows the history wait
      And the chat's history is read and judged after it

  Rule: Pace of history reads
    # No pause when it is off runs as a unit test of the pacer.

    Scenario: history calls keep the pause
      Given a history_pause of 1 second and two watched chats with missed messages
      When the owner starts tgscout and it catches up
      Then every history call to Telegram starts at least 1 second after the one before

  Rule: Catch-up console lines

    Scenario: one line per chat
      Given two watched chats, one with a cursor and one without
      When the owner starts tgscout and it catches up
      Then the console shows a start line and one line per chat, in config order
      And the line of the chat without a cursor says it is a first start

  Rule: Catch-up summary

    Scenario: the summary
      Given three watched chats with history
      When the owner starts tgscout and it catches up
      Then the console shows the summary with chats, messages, matches and the time

    # Judging errors (a failed database write) run as a unit test of MatchMessage:
    # the retries cover every LM Studio failure, so none is left to force from outside.

    Scenario: the summary waits for LM Studio
      Given a watched chat with history and LM Studio that does not answer
      When the owner starts tgscout and it catches up
      Then the console shows no summary while LM Studio is away
      And those messages are not marked processed
