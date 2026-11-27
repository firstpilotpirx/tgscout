@message_matching @e2e
Feature: message-matching — how a new message of a watched chat becomes a decision per request

  Rule: Listening to the watched chats

    Scenario: one message, two requests of its chat
      Given two requests bound to the same watched chat
      When a new message appears in that chat
      Then the LLM is asked once per request
      And two decisions are saved, one per request

  Rule: Pre-filter
    # A service message, a message with no text and a photo caption run in the
    # slice feature of the collector (design § Test levels).

    Scenario: a processed message is skipped
      Given a message that is already marked processed
      When the same message arrives again
      Then the LLM is not asked again and no second match line appears

    Scenario: the same text in another message is judged again
      Given a processed message with some text
      When another message with the same text appears in the same chat
      Then the LLM is asked about the new message

  Rule: Asking the LLM

    Scenario: the LLM is asked with the request and the message
      Given a request "a desk lamp with a clamp" bound to a watched chat and the model example-model
      When a new message "Desk lamp, clamp mount, 10 €" appears in that chat
      Then the LLM is called with the model example-model
      And the call holds the request text and the message text apart from the fixed matching rules

  Rule: Unclear answers

    Scenario: an unclear answer is asked again, then saved as no
      Given the LLM answers neither yes nor no, twice
      When a new message appears in a watched chat
      Then the LLM is asked twice about that request
      And the decision is saved as no match
      And the console shows a warning line with the raw answer

  Rule: Decisions and matches

    Scenario: a matching message is shown on the console
      Given a request "a kids' bike, 16 inch, under 100 €" bound to a watched chat, and the LLM answers yes
      When a new message "Selling a 16-inch kids' bike, 80 €" appears in that chat
      Then the console shows a match line with the request, the chat and the link to the message when the chat has one
      And the decision is saved as a match, the message is marked processed and the cursor moves to it

    Scenario: a message that does not match
      Given the LLM answers no
      When a new message appears in a watched chat
      Then the console shows no match line
      And the decision is saved as no match and the message is marked processed

  Rule: Processed mark and cursor
    # The out-of-order finish runs as unit tests of MessageTracker and
    # MatchMessage: the order of two workers cannot be forced from outside.

  Rule: LLM failure

    Scenario: LM Studio is down
      Given LM Studio does not answer
      When a new message appears in a watched chat
      Then the console shows a line that LM Studio is not answering
      And the message is not marked processed and the cursor does not move past it

  Rule: Edited messages
    # A match saved but not shown runs as a unit test of MatchMessage: the
    # crash between save and send cannot be forced from outside.

    Scenario: an edited message is judged again only where it was «no»
      Given a processed message with a «yes» for one request and a «no» for another
      When the message is edited
      Then the LLM is asked only about the request with «no»
      And no second match line appears for the other

  Rule: Queue capacity
    # A full queue runs in the slice feature of the matcher (design § Test levels).
