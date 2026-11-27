## MODIFIED Requirements

### Requirement: Decisions and matches

For each request asked, tgscout SHALL save a decision with whether it matched and when it was decided. Each «yes» SHALL be sent to the owner as a private Telegram message and shown on the console as one match line followed by the same text as the private message (the owner-notification capability); a saved «yes» decision SHALL be the record of a found message, and nothing else SHALL store matches.

#### Scenario: a matching message is shown on the console

- **GIVEN** a request "a kids' bike, 16 inch, under 100 €" bound to a watched chat, and the LLM answers yes
- **WHEN** a new message "Selling a 16-inch kids' bike, 80 €" appears in that chat
- **THEN** the console shows a match line with the request, the chat and the link to the message when the chat has one
- **AND** the decision is saved as a match, the message is marked processed and the cursor moves to it

#### Scenario: a message that does not match

- **GIVEN** the LLM answers no
- **WHEN** a new message appears in a watched chat
- **THEN** the console shows no match line
- **AND** the decision is saved as no match and the message is marked processed
