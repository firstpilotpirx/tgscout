# tgscout

tgscout watches chosen Telegram chats and writes to you when a message matches a request.

## Install

- Go 1.25.11 or later.
- A C compiler (cgo builds the SQLite driver): Xcode command line tools on macOS, `gcc` on Linux.
- Run once after the clone, so every commit is scanned for secrets:

```bash
git config core.hooksPath .githooks
```

## Run (dev)

```bash
cp tgscout.yaml.example tgscout.yaml
(cd tgscout && go run ./apps/cli --config ../tgscout.yaml)
```

`tgscout --help` lists the options. The database `tgscout.db` is created next to the config; `--db <path>` puts it elsewhere. Press Ctrl+C to stop.

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
