## MODIFIED Requirements

### Requirement: Gap read after a reconnect

When the connection to Telegram is back, tgscout SHALL print one line saying so with the number of chats it reads, read each watched chat again from its cursor the same way catch-up does, with the same pause between history calls, and judge every message of the gap that is not processed yet. Once every chat is read, it SHALL print one line with the number of messages it handed to the judge.

#### Scenario: the connection to Telegram drops

- **GIVEN** tgscout runs and the connection to Telegram drops for a while
- **WHEN** the connection is back
- **THEN** the messages posted during the drop are judged

#### Scenario: a message processed during the drop is not judged twice

- **GIVEN** a message reached tgscout before the connection dropped and was processed during the drop
- **WHEN** the connection is back and tgscout reads the gap
- **THEN** that message is not judged again

#### Scenario: the gap read keeps the pause

- **GIVEN** a `history_pause` of 1 second and two watched chats
- **WHEN** the connection is back and tgscout reads the gap
- **THEN** every history call of the gap read starts at least 1 second after the one before
