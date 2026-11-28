## ADDED Requirements

### Requirement: Stored chats and owner

tgscout SHALL keep, in its database, every chat of the config it found and the owner it found, by their config text. At a later start it SHALL use a stored chat or a stored owner without looking it up by name again, and look up by name only what the config names that the store does not hold. The start SHALL print how many watched chats came from the store and how many it looked up. The store SHALL hold no message text.

#### Scenario: a restart looks up no stored chat by name

- **GIVEN** tgscout found every chat of the config and the owner at an earlier start
- **WHEN** the owner starts tgscout again with the same config
- **THEN** tgscout asks Telegram for no chat and no owner by name
- **AND** it says that every watched chat came from the store

#### Scenario: a chat added to the config is looked up

- **GIVEN** tgscout found two chats at an earlier start and the owner then adds a third chat to the config
- **WHEN** the owner starts tgscout again
- **THEN** tgscout looks up only the third chat by name
- **AND** it says that two chats came from the store and one was looked up

### Requirement: A stored chat that stops working

When Telegram answers a history read of a stored chat that it no longer knows that chat, or that the account cannot read it, tgscout SHALL drop the chat from the store, look it up by name once, print one line saying so and read it again. A chat the new lookup does not find as joined, or finds under another ID, SHALL fail the read as any failed read does, and the next start SHALL look it up by name.

#### Scenario: Telegram no longer knows a stored chat

- **GIVEN** a stored chat that Telegram no longer knows under its stored data, though the account is still in it
- **WHEN** catch-up reads that chat
- **THEN** the console shows one line that tgscout looked the chat up again
- **AND** the chat's history is read and judged

#### Scenario: the account left a stored chat

- **GIVEN** a stored chat the account has left
- **WHEN** catch-up reads that chat
- **THEN** the read of that chat fails and its cursor stays where it was
- **AND** the next start looks the chat up by name and reports that the account has not joined it

## MODIFIED Requirements

### Requirement: Watched chats lookup

After sign-in tgscout SHALL look up every distinct chat of the config that the store does not hold. A chat that does not exist or that the account has not joined SHALL stop the start with one error that lists every such chat with its reason. When Telegram asks to wait during the lookup, tgscout SHALL print how long it waits, wait that time and go on. On success tgscout SHALL print how many chats it watches, then run.

#### Scenario: every chat and the owner are found

- **GIVEN** a config whose chats the account has all joined and whose owner Telegram knows
- **WHEN** the owner starts tgscout
- **THEN** tgscout says it found the owner and watches every chat of the config
- **AND** it goes on running

#### Scenario: a chat the account has not joined

- **GIVEN** a config with one chat the account has not joined and one chat that does not exist
- **WHEN** the owner starts tgscout
- **THEN** tgscout stops with an error that lists both chats, each with its reason
- **AND** it exits with a non-zero code

#### Scenario: Telegram asks to wait during the lookup

- **GIVEN** a config whose chats Telegram looks up only after a short wait
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints how long it waits
- **AND** after the wait it watches every chat of the config

### Requirement: Owner lookup

After sign-in tgscout SHALL find the owner: a stored owner from the store, an owner given as `@username` through Telegram, an owner given as a numeric ID among the account's contacts. At every start it SHALL check that the owner's private chat with tgscout's account is open; when a stored owner's check fails, it SHALL drop the stored owner, find the owner again as if it were not stored and check once more. An owner not found SHALL stop the start with an error that says to add the owner to the account's contacts. A found owner whose private chat with tgscout's account does not exist yet (the owner never wrote to the account) SHALL stop the start with the error «cannot write to the owner» and a hint to write «hi» to tgscout's account once and add it to contacts. A found owner with an open private chat SHALL be printed as written in the config.

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

#### Scenario: a stored owner whose chat was closed

- **GIVEN** a stored owner whose private chat with tgscout's account no longer exists
- **WHEN** the owner starts tgscout
- **THEN** tgscout finds the owner again by name
- **AND** it stops with the error that it cannot write to the owner
