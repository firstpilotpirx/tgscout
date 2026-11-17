@message_matching @slice
Feature: message-matching — the collector and the matcher through the real use cases

  Rule: Pre-filter

    Scenario: a service message is dropped
      Given a watched chat
      When someone joins the chat
      Then the LLM is not asked and nothing is saved

    Scenario: a message with no text is dropped
      Given a watched chat
      When a photo without a caption appears in it
      Then the LLM is not asked

    Scenario: a photo caption counts as text
      Given a watched chat
      When a photo with a caption appears in it
      Then the LLM is asked about the caption

  Rule: Queue capacity

    Scenario: a full queue holds new messages back
      Given a queue size of 1, one worker and an LLM that answers slowly
      When three new messages appear in a watched chat
      Then all three are judged, in the order they appeared
