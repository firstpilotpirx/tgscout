# catch-up Specification

## Purpose
TBD - created by archiving change catch-up. Update Purpose after archive.

## Requirements

### Requirement: Catch-up at start

Once the start says it is running, tgscout SHALL read the missed history of every watched chat while it already listens to new messages. It SHALL take the chats one after another, in the order the config file lists them, and SHALL hand the messages of each chat to the judge oldest first, through the same path as new messages: the same pre-filter, the same processed check, the same LLM call, the same decisions and the same private message to the owner for each match.

#### Scenario: a match in the history reaches the owner

- **GIVEN** a matching message posted while tgscout was off
- **WHEN** the owner starts tgscout
- **THEN** the owner gets the match as a private message

### Requirement: Where catch-up starts and stops

For a chat with a cursor, tgscout SHALL read the messages after the cursor, up to now. For a chat with no cursor (its first start), it SHALL read the last messages up to the depth. In both cases it SHALL read no further back than the chat's depth: the larger depth of the requests that watch the chat, counted from the start; with both a number of messages and a number of days set, reading SHALL stop at whichever limit comes first.

#### Scenario: first start reads the default depth

- **GIVEN** a watched chat with no cursor and 4 days of history
- **WHEN** the owner starts tgscout
- **THEN** tgscout judges the messages of the last 3 days, at most 500, oldest first

#### Scenario: a restart reads only what it missed

- **GIVEN** a chat whose cursor is at message 100 and 20 newer messages
- **WHEN** the owner starts tgscout
- **THEN** only the 20 newer messages are judged

#### Scenario: the message limit comes first

- **GIVEN** a watched chat with no cursor and 700 messages posted in the last day
- **WHEN** the owner starts tgscout
- **THEN** tgscout judges only the newest 500 of them

### Requirement: Gap after a long pause

A chat whose cursor is older than its depth SHALL have a gap: tgscout SHALL read only the depth, say so on the chat's console line and count the chat in the summary as one with a gap.

#### Scenario: a cursor older than the depth

- **GIVEN** a chat whose cursor is at a message of 12 days ago and newer messages over all those days
- **WHEN** the owner starts tgscout
- **THEN** tgscout judges only the messages of the last 3 days
- **AND** the console line of that chat says it has a gap

### Requirement: Depth per request

A history message SHALL be judged only against the requests of its chat whose own depth covers it: the message is among that request's number of newest messages of the chat and no older than its number of days. A history message no request covers SHALL be skipped.

#### Scenario: a request overrides the depth

- **GIVEN** a request with a catch-up depth of 7 days
- **WHEN** catch-up reads a chat of that request for the first time
- **THEN** it reads back 7 days for that request

#### Scenario: a chat overrides the depth

- **GIVEN** a chat of a request with a catch-up depth of 2,000 messages
- **WHEN** catch-up reads that chat for the first time
- **THEN** it reads up to 2,000 messages

#### Scenario: two requests with different depths on one chat

- **GIVEN** a chat bound to a request with 7 days and a request with 3 days, and a message of 5 days ago
- **WHEN** catch-up reads that chat for the first time
- **THEN** the message is judged against the 7-day request only

### Requirement: Nothing judged twice

A history message that is already processed, or is being judged because it also arrived as a new message, SHALL be skipped: the LLM is not asked again and no second private message is sent. A restart SHALL repeat nothing and lose nothing.

#### Scenario: a processed message is not judged again

- **GIVEN** a message after the cursor that is already marked processed
- **WHEN** catch-up reaches it
- **THEN** it is skipped and no second match is sent

#### Scenario: live and catch-up at once

- **GIVEN** catch-up is reading a chat
- **WHEN** a new message appears in that chat
- **THEN** that message is judged once and the owner gets at most one match per request for it

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

### Requirement: Telegram asks to wait during a history read

When Telegram asks to wait (FLOOD_WAIT) during a history read, tgscout SHALL show the wait on the console, wait it out and go on with the same read.

#### Scenario: Telegram asks to wait during catch-up

- **GIVEN** Telegram asks to wait 1 second on the first history read
- **WHEN** the owner starts tgscout
- **THEN** the console shows the wait
- **AND** the chat's history is read and judged after it

### Requirement: Catch-up console lines

tgscout SHALL print one console line when catch-up starts (the number of chats, and that live listening is on) and one line per chat as it is read: the chat's title, or its config reference when it has no title, the number of messages read, and whether it read from the cursor, read a first start's depth or had a gap.

#### Scenario: one line per chat

- **GIVEN** two watched chats, one with a cursor and one without
- **WHEN** the owner starts tgscout
- **THEN** the console shows a start line and one line per chat, in config order
- **AND** the line of the chat without a cursor says it is a first start

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
