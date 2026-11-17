## ADDED Requirements

### Requirement: LLM retries

When a call to LM Studio fails, tgscout SHALL try it again after a pause that starts at 1 second and doubles after each failed try, up to `retry_max_pause` (default 60 seconds). A call that gets no answer (no connection, the timeout, or a server error) SHALL be tried again without end. While a message waits for its retry, it SHALL stay unprocessed, and its chat's cursor SHALL not move past it.

#### Scenario: LM Studio goes away and comes back

- **GIVEN** LM Studio stops answering while messages arrive
- **WHEN** LM Studio answers again
- **THEN** every waiting message is judged and every match reaches the owner, none lost

#### Scenario: the pauses grow up to the cap

- **GIVEN** LM Studio gives no answer to five tries in a row and a retry cap of 4 seconds
- **WHEN** tgscout judges a message
- **THEN** it pauses 1, 2, 4 and 4 seconds between the tries

### Requirement: A message the model never manages

A call that gets an error answer about the message itself (a client error, or an answer tgscout cannot read) SHALL be tried at most 10 times. After the 10th failed try, tgscout SHALL save the decision as «no», print a warning line that names the request, the chat and the message, and go on, so one bad message never stalls its chat.

#### Scenario: a message the model never manages

- **GIVEN** LM Studio answers every try about one message with an error about that message
- **WHEN** tgscout has tried it 10 times
- **THEN** the decision is saved as «no», the message is marked processed and a warning line names it

#### Scenario: no answer never gives up

- **GIVEN** LM Studio gives no answer to 12 tries in a row, then answers «yes»
- **WHEN** tgscout judges a message
- **THEN** the match reaches the owner and no decision is saved as «no»

### Requirement: LLM status lines

When a call to LM Studio gets no answer and LM Studio was answering before, tgscout SHALL print one warning line saying LM Studio is not answering, the retry pauses and that messages wait in the queue. When a call succeeds after that, tgscout SHALL print one line saying LM Studio is back and how many messages wait in the queue. Further failed tries while LM Studio stays away SHALL print no more lines.

#### Scenario: one line when it goes, one when it is back

- **GIVEN** LM Studio gives no answer to three tries in a row, then answers
- **WHEN** tgscout judges two messages during that time
- **THEN** the console shows one line that LM Studio is not answering and one line that it is back

### Requirement: Send retries

When a send of a match fails for a reason other than Telegram asking to wait, tgscout SHALL print a warning line that names the request, the chat and the message, the reason and the next pause. It SHALL try the send again after a pause that starts at 1 second and doubles up to `retry_max_pause`, without end. Until the send is confirmed, the message SHALL stay unprocessed and its cursor SHALL not move past it. The line SHALL never hold the message text or a text the server sent.

#### Scenario: a send fails, then works

- **GIVEN** the first two sends of a match fail
- **WHEN** tgscout sends the match
- **THEN** the console shows two warning lines, the owner gets the match once and the message is marked processed after the send

### Requirement: Full queue

When the queue of messages waiting for LM Studio is full, tgscout SHALL print one warning line saying the queue is full, with its size, and that reading new messages is paused, and SHALL wait to take more messages from Telegram until a place frees up. It SHALL print the line again only after the queue was empty in between.

#### Scenario: the queue is full

- **GIVEN** LM Studio is down long enough that the queue fills up
- **WHEN** more new messages arrive
- **THEN** the collector waits, and after LM Studio is back every message is judged or is read again by catch-up

### Requirement: Telegram reconnect

When the connection to Telegram drops, tgscout SHALL print one warning line saying the connection is lost and that it reconnects, reconnect by itself, and keep the cursor of every watched chat from moving while the connection is away.

#### Scenario: the cursor waits while the connection is away

- **GIVEN** tgscout runs and the connection to Telegram drops
- **WHEN** a message that reached tgscout before the drop is processed
- **THEN** the console shows one line that the connection is lost
- **AND** the cursor of that chat does not move until the gap is read

### Requirement: Gap read after a reconnect

When the connection to Telegram is back, tgscout SHALL print one line saying so with the number of chats it reads, read each watched chat again from its cursor the same way catch-up does, and judge every message of the gap that is not processed yet. Once every chat is read, it SHALL print one line with the number of messages it handed to the judge.

#### Scenario: the connection to Telegram drops

- **GIVEN** tgscout runs and the connection to Telegram drops for a while
- **WHEN** the connection is back
- **THEN** the messages posted during the drop are judged

#### Scenario: a message processed during the drop is not judged twice

- **GIVEN** a message reached tgscout and was processed just before the connection dropped
- **WHEN** the connection is back and tgscout reads the gap
- **THEN** that message is not judged again

### Requirement: Graceful stop

On the first interrupt (Ctrl+C, SIGINT) or terminate signal (SIGTERM), tgscout SHALL stop taking messages from Telegram and from the queue and print one line with the number of messages in work, the grace time `stop_grace` (default 10 seconds) and that a second Ctrl+C stops at once. Each worker SHALL finish the message it holds within the grace time; a message not finished by then SHALL stay unprocessed.

#### Scenario: the owner stops tgscout during work

- **GIVEN** a worker is judging a message
- **WHEN** the owner presses Ctrl+C
- **THEN** the worker finishes that message or leaves it unprocessed, and tgscout exits with code 0

#### Scenario: a judgment longer than the grace time

- **GIVEN** a grace time of 1 second and LM Studio that takes 5 seconds to answer
- **WHEN** the owner presses Ctrl+C while a message is judged
- **THEN** tgscout exits with code 0 within about 1 second and the message is not marked processed

### Requirement: End of a stop

A pause between retries SHALL end at the stop and leave its message unprocessed. Once the workers are done, tgscout SHALL close the database, print one line saying it stopped and how many messages were left in the queue, and exit with code 0.

#### Scenario: a retry pause ends at the stop

- **GIVEN** LM Studio does not answer and a message waits for its next try
- **WHEN** the owner presses Ctrl+C
- **THEN** tgscout exits with code 0 at once and the message is not marked processed

### Requirement: Second interrupt

A second interrupt or terminate signal while tgscout is stopping SHALL end tgscout at once with exit code 130.

#### Scenario: a second Ctrl+C stops at once

- **GIVEN** tgscout is stopping and waits for a message in work
- **WHEN** the owner presses Ctrl+C again
- **THEN** tgscout ends at once with exit code 130

### Requirement: Crash recovery

When the process is killed, the messages in the queue and in work SHALL stay unprocessed in the database, and their chat's cursor SHALL not be past them. At the next start, catch-up SHALL read them again from the cursor and judge them. A match that was sent but not yet marked when the process was killed SHALL be sent again (the allowed rare repeat); no match SHALL be lost. After a clean stop, a start SHALL send no match again for a message that was processed.

#### Scenario: a crash between send and mark

- **GIVEN** a match was sent to the owner and the message was not yet marked processed
- **WHEN** the process is killed and the owner starts tgscout again
- **THEN** the owner gets that match again, and no match is lost

#### Scenario: a crash with messages in the queue

- **GIVEN** messages are waiting in the queue
- **WHEN** the process is killed and the owner starts tgscout again
- **THEN** every one of those messages is judged after the restart

#### Scenario: a clean restart repeats nothing

- **GIVEN** every message read so far is processed
- **WHEN** the owner stops and starts tgscout
- **THEN** no match is sent again
