## MODIFIED Requirements

### Requirement: Cursor during catch-up

While a chat is being caught up, its cursor SHALL NOT move because of new messages of that chat that finish first; it SHALL move once every history message of that chat is handed to the judge, and never past a message still being judged. When the history read of a chat fails, its cursor SHALL stay where it was for the rest of the run, so the next start reads that history again. A stop or a crash during catch-up SHALL leave the cursor where the next start reads every message not yet processed.

#### Scenario: a new message does not move the cursor past unread history

- **GIVEN** catch-up has not yet handed a chat's history to the judge
- **WHEN** a new message of that chat is judged and marked processed
- **THEN** the cursor of that chat stays where it was until the chat's history is handed over

#### Scenario: a failed history read keeps the cursor

- **GIVEN** the history read of a chat fails
- **WHEN** a new message of that chat is judged and marked processed
- **THEN** the cursor of that chat stays where it was

### Requirement: Catch-up summary

When every chat is read and every history message it handed over is processed or failed, tgscout SHALL print the summary: the time catch-up took, the chats read and how many had a gap, the messages read, skipped and judged, the matches sent and the judging errors. A judging error is a message that failed for a reason the retries do not cover, such as the database; a message waiting for a retry of LM Studio or of a send is not settled yet, so the summary waits for it. A failed message SHALL stay unprocessed, for the next start. The summary SHALL go to the console only. A stop before catch-up ends SHALL print no summary.

#### Scenario: the summary

- **GIVEN** three watched chats
- **WHEN** catch-up has read all of them and every caught-up message is processed
- **THEN** the console shows the summary with chats, messages, matches and the time

#### Scenario: judging errors are counted

- **GIVEN** a history message whose judging fails for a reason the retries do not cover
- **WHEN** catch-up has read its chat
- **THEN** the summary counts that message as an error
- **AND** that message is not marked processed

#### Scenario: the summary waits for LM Studio

- **GIVEN** a watched chat with history and LM Studio that does not answer
- **WHEN** the owner starts tgscout
- **THEN** the console shows no summary while LM Studio is away
- **AND** those messages are not marked processed
