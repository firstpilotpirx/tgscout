## MODIFIED Requirements

### Requirement: Processed mark and cursor

tgscout SHALL mark a message processed only after every request of its chat was decided and every «yes» was sent to the owner and confirmed by Telegram; then it SHALL move the chat's cursor, except while that chat is being caught up: then the cursor SHALL wait until every history message of the chat is handed to the judge. The cursor of a chat SHALL move only past messages that are all processed: with several messages judged at once and finishing out of order, it SHALL never pass a message that is not processed yet.

#### Scenario: the cursor never passes an unprocessed message

- **GIVEN** messages 10, 11 and 12 of one chat in the queue and two workers
- **WHEN** message 12 is processed before message 11
- **THEN** the cursor of the chat stays at 10 until message 11 is processed, then moves to 12
