## MODIFIED Requirements

### Requirement: A match goes to the owner

For every «yes» of a request on a message, tgscout SHALL send one private Telegram message from its own account to the owner, and SHALL show the same text on the console under one match line once Telegram confirmed the send. A message that matches several requests SHALL give one private message per request.

#### Scenario: a match reaches the owner

- **GIVEN** a request bound to a watched chat and the LLM answers yes
- **WHEN** a new matching message appears in that chat
- **THEN** the owner gets one private message with the request, the chat and the link to the message when the chat has one
- **AND** the console shows the same match

#### Scenario: two requests match one message

- **GIVEN** two requests of one chat and the LLM answers yes to both
- **WHEN** a new message appears in that chat
- **THEN** the owner gets two private messages, one per request

### Requirement: Text of the private message

The private message SHALL be plain text with no buttons, one part per line: `Match: <request id> — <request text>`, `Chat: <chat title>` (the config text when Telegram gives no title), the link to the message when the chat has one, then the message text. A text longer than 1,000 characters SHALL be cut to 1,000 characters and «…»; the other parts SHALL always stay whole.

#### Scenario: a long message text is cut

- **GIVEN** a request bound to a watched chat and the LLM answers yes
- **WHEN** a matching message with a text of 1,500 characters appears in that chat
- **THEN** the private message holds the first 1,000 characters of the text followed by «…»
- **AND** it still holds the request, the chat and the whole link

### Requirement: Link to the message

A chat with a username SHALL link as `https://t.me/<username>/<message id>`. A supergroup or a channel without a username SHALL link as `https://t.me/c/<chat id>/<message id>`. A basic group (an old-style group that is not a supergroup) SHALL get no link line, since Telegram opens no message link for it; the console match shows the same text.

#### Scenario: a chat without a username

- **GIVEN** a request bound to a watched supergroup that has no username, and the LLM answers yes
- **WHEN** a new matching message appears in that chat
- **THEN** the private message links to the message in the form for members of the chat

#### Scenario: a basic group has no link line

- **GIVEN** a request bound to a watched basic group, and the LLM answers yes
- **WHEN** a new matching message appears in that group
- **THEN** the private message holds the request, the chat and the message text, and no link line
- **AND** the console shows the same text
