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
5. Start tgscout once from a terminal. It asks for the phone number, the login code and the two-step password (hidden), then saves the session.

The session file is `tgscout.session` next to the config, or `TG_SESSION_FILE`; it is written with mode 600 and refused when others can read it (`chmod 600 <file>`). Later starts sign in from it without a question, so tgscout can then run as a service. If Telegram no longer accepts the session, delete the file and start from a terminal again.

## Run (dev)

```bash
cp tgscout.yaml.example tgscout.yaml
(cd tgscout && go run ./apps/cli --config ../tgscout.yaml)
```

`tgscout --help` lists the options. The database `tgscout.db` is created next to the config; `--db <path>` puts it elsewhere. Press Ctrl+C to stop.

### While it runs

After `running`, tgscout judges every new or edited message of the watched chats with the LLM, once per request of that chat. Matches go to stdout, warnings and errors to stderr (made-up examples):

```text
✓ match  14:02  kids-bike  @made_up_market  https://t.me/made_up_market/4211
         Selling a 16-inch kids' bike, 80 €, pick-up in the old town…
⚠ unclear LLM answer for kids-bike on @made_up_market/4215, saved as no
  answer: "It might be, the size is not stated"
✗ LLM failed on @made_up_market/4216: cannot reach http://localhost:1234/v1
  the message stays unprocessed; is LM Studio running?
```

- A match line shows the message time, the request id, the chat as the config writes it, the link and the start of the text (80 characters).
- An answer that is neither yes nor no is asked once more, then saved as «no» with a warning.
- When the LLM call fails, the message stays unprocessed and the chat's cursor does not pass it.
- An edited message is judged again only for the requests that had no match on it.

## Config

`tgscout.yaml` holds what tgscout looks for; [tgscout.yaml.example](tgscout.yaml.example) shows every field. It is read once, at start, and is never committed (`.gitignore`).

- `owner` (required): the Telegram user tgscout writes to, `@username` or a numeric ID.
- `workers` (default 2): how many messages the LLM judges at once; `queue_size` (default 1000): how many messages wait for the LLM; when the queue is full, tgscout waits before it takes more from Telegram and drops none.
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
