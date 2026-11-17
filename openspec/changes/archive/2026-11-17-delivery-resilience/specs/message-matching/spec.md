## MODIFIED Requirements

### Requirement: Processed mark and cursor

tgscout SHALL mark a message processed only after every request of its chat was decided and every «yes» was sent to the owner and confirmed by Telegram; then it SHALL move the chat's cursor, except while that chat is being caught up, and while the connection to Telegram is away until the gap is read: then the cursor SHALL wait until every history message of the chat is handed to the judge. The cursor of a chat SHALL move only past messages that are all processed: with several messages judged at once and finishing out of order, it SHALL never pass a message that is not processed yet.

#### Scenario: the cursor never passes an unprocessed message

- **GIVEN** messages 10, 11 and 12 of one chat in the queue and two workers
- **WHEN** message 12 is processed before message 11
- **THEN** the cursor of the chat stays at 10 until message 11 is processed, then moves to 12

### Requirement: LLM failure

When the LLM call fails (LM Studio down, an error answer, the timeout), tgscout SHALL try it again as the delivery capability's LLM retries say. While the message waits for a retry, it SHALL stay unprocessed, its chat's cursor SHALL not move past it, and the other workers SHALL go on with the next messages.

#### Scenario: LM Studio is down

- **GIVEN** LM Studio does not answer
- **WHEN** a new message appears in a watched chat
- **THEN** the console shows a line that LM Studio is not answering
- **AND** the message is not marked processed and the cursor does not move past it
