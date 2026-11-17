## MODIFIED Requirements

### Requirement: Owner lookup

After sign-in tgscout SHALL find the owner: an owner given as `@username` through Telegram, an owner given as a numeric ID among the account's contacts. An owner not found SHALL stop the start with an error that says to add the owner to the account's contacts. A found owner whose private chat with tgscout's account does not exist yet (the owner never wrote to the account) SHALL stop the start with the error «cannot write to the owner» and a hint to write «hi» to tgscout's account once and add it to contacts. A found owner with an open private chat SHALL be printed as written in the config.

#### Scenario: the owner is not found

- **GIVEN** a config whose owner is a numeric ID that is not among the account's contacts
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that says to add the owner to the account's contacts
- **AND** it exits with a non-zero code

#### Scenario: the owner never wrote to tgscout's account

- **GIVEN** a config whose owner Telegram knows but who never wrote to tgscout's account
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that it cannot write to the owner and a hint to write «hi» to its account once
- **AND** it exits with a non-zero code
