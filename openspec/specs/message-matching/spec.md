# message-matching Specification

## Purpose
How a new message in a watched chat becomes a decision per request of that chat: what is dropped before the LLM, how the LLM is asked, how a match is shown on the console, and when a message counts as processed and the chat's cursor moves.

## Requirements

### Requirement: Listening to the watched chats

Once the start says it is running, tgscout SHALL take every new message and every edited message of each watched chat as it appears in Telegram, and judge it against every request that watches that chat; a request SHALL never be judged on a message of a chat it does not watch.

#### Scenario: one message, two requests of its chat

- **GIVEN** two requests bound to the same watched chat
- **WHEN** a new message appears in that chat
- **THEN** the LLM is asked once per request
- **AND** two decisions are saved, one per request

### Requirement: Pre-filter

Before the LLM, tgscout SHALL drop a service message (a join, a pin, a title change), a message with no text, and a message it saw already; a photo or a file with a caption SHALL be judged by its caption. A message seen already SHALL be the same chat and message ID seen twice, whether it is still being judged or already processed; the same text in another message SHALL be judged again. A dropped message SHALL save nothing and ask the LLM nothing.

#### Scenario: a service message is dropped

- **GIVEN** a watched chat
- **WHEN** someone joins the chat
- **THEN** the LLM is not asked and nothing is saved

#### Scenario: a message with no text is dropped

- **GIVEN** a watched chat
- **WHEN** a photo without a caption appears in it
- **THEN** the LLM is not asked

#### Scenario: a photo caption counts as text

- **GIVEN** a watched chat
- **WHEN** a photo with a caption appears in it
- **THEN** the LLM is asked about the caption

#### Scenario: a processed message is skipped

- **GIVEN** a message that is already marked processed
- **WHEN** the same message arrives again
- **THEN** the LLM is not asked again and no second match line appears

#### Scenario: the same text in another message is judged again

- **GIVEN** a processed message with some text
- **WHEN** another message with the same text appears in the same chat
- **THEN** the LLM is asked about the new message

### Requirement: Asking the LLM

tgscout SHALL ask the LLM at the configured LM Studio address with the configured model and timeout, once per request of the message's chat, with the fixed matching rules kept apart from the request text and the message text, and SHALL take a strict yes or no from the answer.

#### Scenario: the LLM is asked with the request and the message

- **GIVEN** a request "a desk lamp with a clamp" bound to a watched chat and the model example-model
- **WHEN** a new message "Desk lamp, clamp mount, 10 €" appears in that chat
- **THEN** the LLM is called with the model example-model
- **AND** the call holds the request text and the message text apart from the fixed matching rules

### Requirement: Unclear answers

An LLM answer that is neither yes nor no SHALL be asked once more with a stricter instruction; a second unclear answer SHALL be saved as «no», and the console SHALL show a warning line with the request, the message and the raw answer.

#### Scenario: an unclear answer is asked again, then saved as no

- **GIVEN** the LLM answers neither yes nor no, twice
- **WHEN** a new message appears in a watched chat
- **THEN** the LLM is asked twice about that request
- **AND** the decision is saved as no match
- **AND** the console shows a warning line with the raw answer

### Requirement: Decisions and matches

For each request asked, tgscout SHALL save a decision with whether it matched and when it was decided. Each «yes» SHALL be shown on the console as one match line with the time, the request id, the chat, a link to the message and the start of its text; a saved «yes» decision SHALL be the record of a found message, and nothing else SHALL store matches.

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

tgscout SHALL mark a message processed only after every request of its chat was decided and every «yes» was shown; then it SHALL move the chat's cursor. The cursor of a chat SHALL move only past messages that are all processed: with several messages judged at once and finishing out of order, it SHALL never pass a message that is not processed yet.

#### Scenario: the cursor never passes an unprocessed message

- **GIVEN** messages 10, 11 and 12 of one chat in the queue and two workers
- **WHEN** message 12 is processed before message 11
- **THEN** the cursor of the chat stays at 10 until message 11 is processed, then moves to 12

### Requirement: LLM failure

When the LLM call fails (LM Studio down, an error answer, the timeout), tgscout SHALL print an error line that names the chat and the message, leave the message unprocessed, keep the cursor from moving past it, and go on with the next messages.

#### Scenario: LM Studio is down

- **GIVEN** LM Studio does not answer
- **WHEN** a new message appears in a watched chat
- **THEN** the console shows an error line
- **AND** the message is not marked processed and the cursor does not move past it

### Requirement: Edited messages

An edited message SHALL be judged again only for the requests of its chat that have no «yes» shown yet; a request whose «yes» was shown SHALL not be asked again, and a shown «yes» SHALL never be shown twice for the same message and request. A «yes» that was saved but not shown (the console write failed, or tgscout stopped in between) SHALL be judged and shown again, so no match is lost. Judging an edit SHALL not move the cursor.

#### Scenario: an edited message is judged again only where it was «no»

- **GIVEN** a processed message with a «yes» for one request and a «no» for another
- **WHEN** the message is edited
- **THEN** the LLM is asked only about the request with «no»
- **AND** no second match line appears for the other

#### Scenario: a match saved but not shown is shown again

- **GIVEN** a message with a saved «yes» for a request that was never shown
- **WHEN** the message is judged again
- **THEN** the match line for that request appears

### Requirement: Queue capacity

Messages waiting for the LLM SHALL wait in a queue that holds at most `queue_size` messages; when the queue is full, tgscout SHALL wait to take more messages from Telegram until a place frees up, and drop none.

#### Scenario: a full queue holds new messages back

- **GIVEN** a queue size of 1, one worker and an LLM that answers slowly
- **WHEN** three new messages appear in a watched chat
- **THEN** all three are judged, in the order they appeared
