## ADDED Requirements

### Requirement: A match goes to the owner

For every «yes» of a request on a message, tgscout SHALL send one private Telegram message from its own account to the owner, and SHALL show the same text on the console under one match line once Telegram confirmed the send. A message that matches several requests SHALL give one private message per request.

#### Scenario: a match reaches the owner

- **GIVEN** a request bound to a watched chat and the LLM answers yes
- **WHEN** a new matching message appears in that chat
- **THEN** the owner gets one private message with the request, the chat and a link to the message
- **AND** the console shows the same match

#### Scenario: two requests match one message

- **GIVEN** two requests of one chat and the LLM answers yes to both
- **WHEN** a new message appears in that chat
- **THEN** the owner gets two private messages, one per request

### Requirement: Text of the private message

The private message SHALL be plain text with no buttons, one part per line: `Match: <request id> — <request text>`, `Chat: <chat title>` (the chat as the config writes it when Telegram gives no title), the link to the message, then the message text. A text longer than 1,000 characters SHALL be cut to its first 1,000 characters and «…»; the other parts SHALL always stay whole. A chat with a username SHALL link as `https://t.me/<username>/<message id>`, one without as `https://t.me/c/<chat id>/<message id>`.

#### Scenario: a long message text is cut

- **GIVEN** a request bound to a watched chat and the LLM answers yes
- **WHEN** a matching message with a text of 1,500 characters appears in that chat
- **THEN** the private message holds the first 1,000 characters of the text followed by «…»
- **AND** it still holds the request, the chat and the whole link

### Requirement: Send before the processed mark

tgscout SHALL count a «yes» as sent only after Telegram confirmed its private message; only then SHALL it record the send, and only after every request of the message was decided and every «yes» sent SHALL it mark the message processed and move the cursor. When tgscout stops before Telegram confirmed a send, the message SHALL stay unprocessed and the cursor SHALL not move past it, so the match is sent again at the next start; such a rare repeat is allowed, a lost match is not.

#### Scenario: the message is marked processed only after the send

- **GIVEN** Telegram has not yet confirmed the send of a match
- **WHEN** tgscout stops at that moment
- **THEN** the message is not marked processed and the cursor has not moved past it

### Requirement: Telegram asks to wait before a send

When Telegram answers a send with a wait (FLOOD_WAIT), tgscout SHALL print a warning line with how long it waits, wait that long with no upper limit, then send again; it SHALL mark the message processed only after the send is confirmed. A stop during the wait SHALL leave the message unprocessed.

#### Scenario: Telegram asks to wait

- **GIVEN** Telegram answers a send with a wait of 5 seconds
- **WHEN** tgscout sends a match
- **THEN** it waits 5 seconds, sends again and marks the message processed after the send is confirmed

### Requirement: A failed send

When a send fails for any other reason (no network, Telegram refuses it), tgscout SHALL print an error line that names the chat and the message, leave the message unprocessed, keep the cursor from moving past it, and go on with the next messages. The error line SHALL never hold the message text or a text the server sent.

#### Scenario: a send fails

- **GIVEN** the send of a match fails
- **WHEN** the matcher handles the message
- **THEN** the message stays unprocessed and the console shows an error line

### Requirement: Writing only to the owner

tgscout SHALL write only to the owner: never to a watched chat and never to anyone else. A private message to tgscout's account, from anyone, the owner included, SHALL get no answer and SHALL not be judged; version 1 has no commands. tgscout SHALL send no message other than the matches (no message at start).

#### Scenario: tgscout writes only to the owner

- **GIVEN** someone other than the owner writes to tgscout's account
- **WHEN** tgscout runs
- **THEN** tgscout does not answer and writes nothing to that person
