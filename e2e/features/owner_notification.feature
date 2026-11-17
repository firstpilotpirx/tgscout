@owner_notification @e2e
Feature: owner-notification — each match goes to the owner as a private Telegram message

  Rule: A match goes to the owner

    Scenario: a match reaches the owner
      Given a request bound to a watched chat and the LLM answers yes
      When a new matching message appears in that chat
      Then the owner gets one private message with the request, the chat and a link to the message
      And the console shows the same match

    Scenario: two requests match one message
      Given two requests of one chat and the LLM answers yes to both
      When a new message appears in that chat
      Then the owner gets two private messages, one per request

  Rule: Text of the private message
    # A long text cut at 1,000 characters runs as a unit test of the text
    # (design § Test levels): the console and Telegram get the same text.

  Rule: Link to the message
    # The link of a chat without a username runs as a unit test of the collector.

  Rule: Send before the processed mark
    # The stop during a send runs in the slice feature of the matcher: it cannot
    # be timed from outside the binary.

  Rule: Telegram asks to wait before a send
    # The 5-second wait runs in the slice feature of the matcher with an
    # injected sleep, so the suite does not wait 5 seconds.

  Rule: A failed send

    Scenario: a send fails
      Given every send of a match fails
      When the matcher handles the message
      Then the message stays unprocessed and the console shows a warning line for each failed try

  Rule: Writing only to the owner

    Scenario: tgscout writes only to the owner
      Given someone other than the owner writes to tgscout's account
      When tgscout runs
      Then tgscout does not answer and writes nothing to that person
