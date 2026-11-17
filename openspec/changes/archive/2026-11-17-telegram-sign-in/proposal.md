## Why

tgscout reads chats and writes to the owner as one ordinary Telegram account of its own (a second account, not the owner's) over MTProto — by design no Bot API, because a bot joins a group only when an admin adds it, and tgscout must read chats where the owner is not an admin. Before any message can be read or sent, tgscout has to sign in as that account, keep the session safe on disk (readable by the owner only), and find every chat of the config and the owner in Telegram. Doing this as its own slice gives the owner a first real run: tgscout signs in and confirms it sees every watched chat.

Design pages this slice implements:

- [L1 System Context](../../../docs/architecture/context.html) — Telegram over MTProto as tgscout's own ordinary account; tgscout answers only the owner's user ID.
- [L3 Component — tgscout § Folders and layers](../../../docs/architecture/tgscout.html) — `infra/telegram`: the MTProto client and session, shared by collect and notify.
- [Start flow](../../../docs/flows/start.html) — «sign in as the reader account».
- [Quality attributes](../../../docs/architecture/quality-attributes.html) — Privacy: the session file is readable by the owner only (mode 600).

## What Changes

- `infra/telegram` holds one MTProto client and its session for the whole process; the collector and the notifier (later slices) share it.
- `TG_API_ID` and `TG_API_HASH` come from `.env` (or the environment); the session lives in a session file whose path `.env` names. None of them is ever printed or logged.
- First start: tgscout asks in the console for the phone number, the login code Telegram sends, and the two-step password when the account has one; then it saves the session file with mode 600.
- Later starts sign in silently from the session file.
- A session file readable by others than the owner stops the start with an error that says how to fix the mode.
- A revoked or broken session stops the start with a message that tells the owner to sign in again (and how).
- After sign-in tgscout looks up every chat of `tgscout.yaml` and the owner: a chat the account has not joined, or that does not exist, stops the start with an error that lists every such chat; the owner not found stops the start too.
- On success tgscout prints which account it signed in as (name only, no phone) and how many chats it watches.
- `README.md` explains how to get `api_id` / `api_hash`, how to prepare the second account, join the chats with it, and add the owner to its contacts.

Current stage: PoC. Target stage: PoC.
Scope: MTProto client and session in `infra/telegram`, interactive first sign-in, session file permissions, chat and owner lookup at start, the README setup section.
Acceptance bar: the scenarios below pass against a fake MTProto server or a faked client port; a manual run by the owner signs in with a real second account, creates the session file with mode 600, and lists the watched chats.
Out of scope: reading messages (`live-matching`); sending messages (`owner-notification`); joining chats for the owner (the owner joins them with tgscout's account); a log-out command; reconnecting after a network drop (`delivery-resilience`).
Interface: CLI — console prompts for phone, code and password at first start; start errors.

## Acceptance scenarios

- Scenario: first start signs in and saves the session
  - Given no session file yet
  - When the owner starts tgscout and types the phone number, the login code and the password
  - Then tgscout signs in and saves the session file readable by the owner only
- Scenario: a later start signs in silently
  - Given a valid session file
  - When the owner starts tgscout
  - Then tgscout signs in without asking anything
- Scenario: a wrong login code
  - Given no session file yet
  - When the owner types a wrong login code
  - Then tgscout says the code is wrong and asks again, and no session file is saved
- Scenario: the session file is open to others
  - Given a session file readable by other users
  - When the owner starts tgscout
  - Then tgscout stops with an error that says how to make the file private
- Scenario: a revoked session
  - Given a session that Telegram no longer accepts
  - When the owner starts tgscout
  - Then tgscout stops and tells the owner to sign in again
- Scenario: a chat the account has not joined
  - Given a config with a chat the account has not joined
  - When the owner starts tgscout
  - Then tgscout stops with an error that lists that chat
- Scenario: secrets never reach the console
  - Given any start, good or failed
  - When tgscout prints its output
  - Then the output holds no phone number, api_hash, login code, password or session data

## Capabilities

### New Capabilities

- `telegram-session`: signing in as tgscout's own Telegram account, keeping its session private, and finding the watched chats and the owner.

## Impact

- New: `tgscout/infra/telegram/`, `.env.example` fields, README setup section.
- Changed: `apps/cli/main.go` (sign-in step of the start flow).
- New dependency: an MTProto library.

## Open points

- MTProto library: the kit tech matrix has no Telegram row. Proposed: `github.com/gotd/td` (pure Go, maintained, no cgo); check the matrix and license at design.
- Prompts when stdin is not a terminal (run as a service): proposed — refuse with «sign in once from a terminal first».
- A chat not joined: stop the start, or warn and skip it. Proposed: stop, because a silently skipped chat hides missed matches.
- Rate limits at lookup of up to 100 chats (FLOOD_WAIT): proposed — wait the time Telegram names and go on, with a console line.

## Release

SemVer: MINOR — tgscout signs in to Telegram.
Target version: 0.3.0.
