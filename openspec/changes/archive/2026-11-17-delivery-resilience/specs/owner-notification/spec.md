## MODIFIED Requirements

### Requirement: A failed send

When a send fails for any other reason (no network, Telegram refuses it), tgscout SHALL print a warning line that names the chat and the message and try the send again as the delivery capability's send retries say. Until the send is confirmed, the message SHALL stay unprocessed and its chat's cursor SHALL not move past it. The warning line SHALL never hold the message text or a text the server sent.

#### Scenario: a send fails

- **GIVEN** every send of a match fails
- **WHEN** the matcher handles the message
- **THEN** the message stays unprocessed and the console shows a warning line for each failed try
