## MODIFIED Requirements

### Requirement: Decisions and matches

For each request asked, tgscout SHALL save a decision with whether it matched and when it was decided. Each «yes» SHALL be sent to the owner as a private Telegram message and shown on the console as one match line followed by the same text as the private message (the owner-notification capability); a saved «yes» decision SHALL be the record of a found message, and nothing else SHALL store matches.

#### Scenario: a matching message is shown on the console

- **GIVEN** a request "a kids' bike, 16 inch, under 100 €" bound to a watched chat, and the LLM answers yes
- **WHEN** a new message "Selling a 16-inch kids' bike, 80 €" appears in that chat
- **THEN** the console shows a match line with the request, the chat and a link to the message
- **AND** the decision is saved as a match, the message is marked processed and the cursor moves to it

#### Scenario: a message that does not match

- **GIVEN** the LLM answers no
- **WHEN** a new message appears in a watched chat
- **THEN** the console shows no match line
- **AND** the decision is saved as no match and the message is marked processed

### Requirement: Processed mark and cursor

tgscout SHALL mark a message processed only after every request of its chat was decided and every «yes» was sent to the owner and confirmed by Telegram; then it SHALL move the chat's cursor. The cursor of a chat SHALL move only past messages that are all processed: with several messages judged at once and finishing out of order, it SHALL never pass a message that is not processed yet.

#### Scenario: the cursor never passes an unprocessed message

- **GIVEN** messages 10, 11 and 12 of one chat in the queue and two workers
- **WHEN** message 12 is processed before message 11
- **THEN** the cursor of the chat stays at 10 until message 11 is processed, then moves to 12

### Requirement: Edited messages

An edited message SHALL be judged again only for the requests of its chat that have no «yes» sent yet; a request whose «yes» was sent SHALL not be asked again, and a sent «yes» SHALL never be sent twice for the same message and request. A «yes» that was saved but not sent (the send failed, or tgscout stopped in between) SHALL be judged and sent again, so no match is lost. Judging an edit SHALL not move the cursor.

#### Scenario: an edited message is judged again only where it was «no»

- **GIVEN** a processed message with a «yes» for one request and a «no» for another
- **WHEN** the message is edited
- **THEN** the LLM is asked only about the request with «no»
- **AND** no second match line appears for the other

#### Scenario: a match saved but not shown is shown again

- **GIVEN** a message with a saved «yes» for a request that was never sent
- **WHEN** the message is judged again
- **THEN** the match for that request is sent and its match line appears
