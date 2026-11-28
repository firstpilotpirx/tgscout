## ADDED Requirements

### Requirement: Pace of history reads

tgscout SHALL keep at least the `history_pause` of the config between the starts of two history calls to Telegram, across every chat it reads, so catch-up never sends its calls in a burst. A history call repeated after a FLOOD_WAIT SHALL keep the pause too.

#### Scenario: history calls keep the pause

- **GIVEN** a `history_pause` of 1 second and two watched chats with missed messages
- **WHEN** catch-up reads both chats
- **THEN** every history call to Telegram starts at least 1 second after the one before

#### Scenario: no pause when it is off

- **GIVEN** a `history_pause` of 0 seconds
- **WHEN** catch-up reads the watched chats
- **THEN** the history calls go one after another with no pause

## MODIFIED Requirements

### Requirement: Catch-up console lines

tgscout SHALL print one console line when catch-up starts (the number of chats, that live listening is on, and the pause between history reads when it is not off) and one line per chat as it is read: the chat's title, or its config reference when it has no title, the number of messages read, and whether it read from the cursor, read a first start's depth or had a gap.

#### Scenario: one line per chat

- **GIVEN** two watched chats, one with a cursor and one without
- **WHEN** the owner starts tgscout
- **THEN** the console shows a start line and one line per chat, in config order
- **AND** the line of the chat without a cursor says it is a first start

#### Scenario: the start line names the pause

- **GIVEN** a `history_pause` of 1 second
- **WHEN** catch-up starts
- **THEN** the start line says that history reads keep 1 second between them
