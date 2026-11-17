# tgscout

tgscout watches chosen Telegram chats and writes to you when a message matches a request.

## Install

- Go 1.25.11 or later; the go command then downloads the pinned toolchain (`toolchain go1.27.4`).
- A C compiler (cgo builds the SQLite driver): Xcode command line tools on macOS, `gcc` on Linux.
- Run once after the clone, so every commit is scanned for secrets:

```bash
git config core.hooksPath .githooks
```

## Setup

tgscout reads chats from its own Telegram account, a second account, not yours.

1. Get an `api_id` and an `api_hash` at [my.telegram.org](https://my.telegram.org) → API development tools, signed in as tgscout's account.
2. Copy [.env.example](.env.example) to `.env` and fill in `TG_API_ID` and `TG_API_HASH`. `.env` is never committed (`.gitignore`); a variable set in the environment wins over `.env`.
3. With tgscout's account, join every chat the config names; a chat it has not joined stops the start.
4. Add yourself (the `owner` of the config) to the contacts of tgscout's account, so a numeric owner ID can be found.
5. First contact: from your own account, write «hi» to tgscout's account once. Telegram lets an account write first only to someone who wrote to it, so without this message tgscout stops at the start with «cannot write to the owner».
6. Start tgscout once from a terminal. It asks for the phone number, the login code and the two-step password (hidden), then saves the session.

The session file is `tgscout.session` next to the config, or `TG_SESSION_FILE`; it is written with mode 600 and refused when others can read it (`chmod 600 <file>`). Later starts sign in from it without a question, so tgscout can then run as a service. If Telegram no longer accepts the session, delete the file and start from a terminal again.

## Run (dev)

```bash
cp tgscout.yaml.example tgscout.yaml
(cd tgscout && go run ./apps/cli --config ../tgscout.yaml)
```

`tgscout --help` lists the options. The database `tgscout.db` is created next to the config; `--db <path>` puts it elsewhere. Press Ctrl+C to stop (see [Stop and crash](#stop-and-crash)).

### Catch-up at start

While live listening is already on, tgscout reads what each watched chat posted while it was off, chat by chat in config order. It reads from the chat's cursor (the last processed message); on a first start, or after a pause longer than the depth, it reads only the depth (`catch_up` in the config). Each history match goes to you as its own private message; the summary goes only to the console (made-up example):

```text
→ catch-up started · 3 chats · live listening is on
✓ catch-up: Riverside flea market · 214 messages from the cursor
✓ catch-up: Kids' things swap · 500 messages · first start, last 3 days
⚠ catch-up: Bikes for sale · 500 messages · off for 12 days, read only the last 3 days
✓ catch-up done in 4m35s
  ✓ chats      3  (1 with a gap)
  ✓ messages   1,042 read · 18 skipped · 1,024 judged
  ● matches    2  (sent to you in Telegram)
  ✓ errors     0
```

- A message already processed is skipped; a message that is in the history and also arrives live is judged once.
- A chat's cursor waits until that chat's history is handed to the judge.
- A message whose judging fails for a reason the retries do not cover (the database) stays unprocessed and is counted in «errors»; the next start judges it again. While LM Studio is away, the summary waits for it.

### While it runs

After `running`, tgscout judges every new or edited message of the watched chats with the LLM, once per request of that chat. Each match goes to you, the owner, as a private Telegram message from tgscout's account:

```text
Match: kids-bike — a kids' bike, 16 inch, under 100 €
Chat: Made-up market
https://t.me/made_up_market/4211
Selling a 16-inch kids' bike, 80 €, pick-up in the old town
```

The message text is cut at 1,000 characters and ends with «…». tgscout writes only to the owner and never answers anyone else.

Once Telegram confirms the send, the same text goes to stdout; warnings and errors go to stderr (made-up examples):

```text
✓ match sent  14:02  kids-bike
  Match: kids-bike — a kids' bike, 16 inch, under 100 €
  Chat: Made-up market
  https://t.me/made_up_market/4211
  Selling a 16-inch kids' bike, 80 €, pick-up in the old town
Telegram asks to wait 5s (FLOOD_WAIT), then sends again
⚠ unclear LLM answer for kids-bike on @made_up_market/4215, saved as no
  answer: "It might be, the size is not stated"
```

- A match line shows the message time and the request id, then the text sent to the owner.
- A message is marked processed only after every send of it is confirmed.
- An answer that is neither yes nor no is asked once more, then saved as «no» with a warning.
- An edited message is judged again only for the requests that had no match on it.

### Retries and reconnect

Nothing is dropped when LM Studio or Telegram goes away: the message waits, unprocessed, and its chat's cursor does not pass it (made-up examples):

```text
⚠ LM Studio is not answering · retry in 1s, 2s, 4s … up to 60s · messages wait in the queue
✓ LM Studio is back · 1,018 messages waiting
⚠ send failed: kids-bike on @made_up_market/4529 · cannot reach Telegram · retry in 2s · the message stays unprocessed
⚠ judge gave up after 10 tries: kids-bike on @made_up_market/4610 · saved as no · made-up LM Studio answered 400
⚠ queue is full (1,000) · reading new messages paused
⚠ Telegram connection lost · reconnecting
✓ Telegram connection back · reading 3 chats from their cursors
✓ gap read done · 42 messages to judge
```

- A call to LM Studio that gets no answer (no connection, the timeout, a server error) and a failed send are tried again without end. The pause starts at 1 s and doubles up to `retry_max_pause`.
- A call LM Studio answers with an error about the message itself (a client error, an answer tgscout cannot read) is tried 10 times, then saved as «no» with a warning, so one bad message never stalls its chat.
- When the queue is full, tgscout waits before it takes more from Telegram; the line shows once per full spell.
- When the connection to Telegram drops, tgscout reconnects by itself, then reads each watched chat again from its cursor, as catch-up does.

### Stop and crash

```text
→ stopping · finishing 2 messages in work · up to 10s · Ctrl+C again to stop now
✓ stopped · 18 messages in the queue left unprocessed, catch-up reads them next start
```

- The first Ctrl+C (or SIGTERM) stops taking messages; each worker gets `stop_grace` to finish the message it holds. A message not finished by then, or waiting for a retry, stays unprocessed. tgscout exits with code 0.
- A second Ctrl+C ends tgscout at once with exit code 130.
- After a crash (the process killed), the next start reads every unprocessed message again from the cursor. A match sent but not yet marked can come once more; no match is lost.

## Config

`tgscout.yaml` holds what tgscout looks for; [tgscout.yaml.example](tgscout.yaml.example) shows every field. It is read once, at start, and is never committed (`.gitignore`).

- `owner` (required): the Telegram user tgscout writes to, `@username` or a numeric ID.
- `workers` (default 2): how many messages the LLM judges at once; `queue_size` (default 1000): how many messages wait for the LLM; when the queue is full, tgscout waits before it takes more from Telegram and drops none.
- `retry_max_pause` (default `60s`): the longest pause between two tries of LM Studio or of a send; `stop_grace` (default `10s`): how long a stop waits for the messages in work.
- `lm_studio.url` (default `http://localhost:1234/v1`), `lm_studio.model` (required), `lm_studio.timeout` (default `60s`).
- `catch_up.messages` and `catch_up.days` (defaults 500 and 3): how far back each chat is read at start.
- `requests`: each with a unique `id`, the `find` text and one or more `chats` (`@username`, a public `t.me` link or a numeric ID).
- Only `catch_up` can be set again on a request or on one chat of a request; the nearest level wins (chat, then request, then root), field by field.

On a good start tgscout prints one line: requests, chats and model. A bad config lists every problem with its line and stops with exit code 1. Secrets never go into this file.

## Run (production)

```bash
(cd tgscout && go build -o ../bin/tgscout ./apps/cli)
./bin/tgscout --config /path/to/tgscout.yaml
```

Run it under a supervisor that sends SIGTERM to stop it; tgscout closes the database and exits with code 0.

## Test

```bash
scripts/qa.sh quick
```

```bash
scripts/qa.sh
```

`quick` runs vet, lint, the architecture check and the unit tests. The default `full` level adds the vulnerability scan, complexity, typos, shellcheck, duplicates, integration tests, the godog E2E scenarios in `e2e/` and `gitleaks`. Every pinned tool must be installed at its version; `qa.sh` prints the install command of a missing one.

## License

Licensed under either of

- Apache License, Version 2.0 ([LICENSE-APACHE](LICENSE-APACHE) or <https://www.apache.org/licenses/LICENSE-2.0>)
- MIT license ([LICENSE-MIT](LICENSE-MIT) or <https://opensource.org/licenses/MIT>)

at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted for inclusion in the work by you, as defined in the Apache-2.0 license, shall be dual licensed as above, without any additional terms or conditions.
