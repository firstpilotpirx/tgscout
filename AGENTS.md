# tgscout — AI agent instructions

## What this project is

tgscout reads chosen Telegram chats from its own ordinary Telegram account (MTProto, a second account, not the owner's) and writes to the owner from that account when a message matches a standing request ("a kids' bike, 16 inch, under 100 €"). A local LLM judge (LM Studio through its OpenAI-compatible API) decides each match. Version 1 is a console program: real time plus a short catch-up of recent history. Written in Go; the code lives in `tgscout/` (Go workspace, module `github.com/firstpilotpirx/tgscout`) with the layout of `~/.claude/rules-custom/architecture-go.md`; the E2E tests are the separate module `e2e/`.

**Product lifecycle stage:** PoC.

## Rules and sources of truth

`CLAUDE.md` is a link to this file: Claude Code and every other agent read the same text.

- Engineering rules and skills are global, from agent-kit (`~/.claude/`): the index `~/.claude/rules/index.md` is always loaded; each rule in `~/.claude/rules-custom/` is read when its trigger fires.
- **Primary language:** Go. Use the `~/.claude/rules-custom/*-go.md` variants wherever a rule links to a per-language file.
- **Sources of truth, in order:** (1) the OpenSpec artifacts under `openspec/`; (2) the code. Nothing else outranks them.
- **No glossary.** A term that needs defining is defined in the spec that uses it.

## Workflow — OpenSpec

Every feature, bug fix and behavior change runs through the stages below, in order. Changes without behavior change (typos, formatting, renames, docs, dependency bumps) skip stages 1–3 and 6–7; they still go through a branch, the checks, a PR and a merge.

| # | Stage | How | Done when |
|---|---|---|---|
| 1 | Explore (always) | `/opsx:explore` | the idea is clear and the scope settled; no files written |
| 2 | Propose | `/opsx:new`, then `/opsx:continue` per artifact (proposal → specs → design → tasks) | `kit proposal-lint <name>` and `openspec validate <name>` pass; the `plan-reviewer` agent answers `RESULT: PASS` |
| 3 | Approval | show the proposal to the user | the user approved it — **stop and wait**; no code before this |
| 4 | Branch | `kit branch <type>/<name>` (`feat`, `fix`, `chore`) | |
| 5 | Apply | `/opsx:apply`, test-first (§ TDD) | every task in `tasks.md` is checked; one commit per task group (`kit ship`) |
| 6 | Verify | `/opsx:verify`; then the `spec-test-reviewer` agent once, only for a change that is not small | no CRITICAL issues; for a reviewed change, `RESULT: PASS` |
| 7 | Archive | `/opsx:archive`, then commit | specs merged into `openspec/specs/`, change in `openspec/changes/archive/` |
| 8 | Full check | `kit verify` (§ How to verify a change) | passes on the final commit |
| 9 | Pull request | `kit pr` | the PR body lists the checks that ran |
| 10 | Manual run | the user, only for the areas `manual_run` in `.claude/kit.yaml` lists | the user confirmed it works |
| 11 | Merge | `kit land` | the change is on `main` and the main checkout is updated |
| 12 | Release | the release command of § Release, when `kit release-needed` names a version | tag `vX.Y.Z` and its release exist |

- Stages 4–12 are pre-authorized: commit, push, open the PR, squash-merge, tag and release without asking.
- Skills (agent-kit): `/create-change` files a change for later; `/proceed-change <name>` runs every stage, stopping only at the gates picked at start.

## Open source and secrets

The repository is public on GitHub (`firstpilotpirx/tgscout`), dual-licensed `MIT OR Apache-2.0`. Nothing private ever enters it: not in files, commits, OpenSpec artifacts, tests, fixtures, logs or PR text.

- **Never commit:** Telegram session files or strings, `api_id` / `api_hash`, bot tokens, LLM API keys, phone numbers, user or chat IDs of real people, real chat messages or media, the local database, judge decision logs from real chats, personal or work data.
- Secrets live only in `.env` (ignored by git); the repository holds `.env.example` with empty values once the code needs one.
- Test data and fixtures are made up; never copy a real message, name or chat into them.
- Before every commit and in CI, `gitleaks git` scans the history with `.gitleaks.toml`; a finding stops the commit until it is removed from the history.
- A secret that reached a commit is revoked first (new session, new token), then removed from the history.

## TDD

Every behavior change in stage 5 is test-first (`tdd` skill):

1. **Red** — write the test for the new behavior and run only it; it fails because the behavior is missing.
2. **Green** — write the smallest code that makes it pass.
3. **Refactor** — clean up with the test green.

- `tasks.md` lists the Red task before the Green task for each behavior.
- A bug fix starts with a test that reproduces the bug.

## Gherkin

The runner is godog, called from `go test` in the `e2e/` module; `.feature` files live in `e2e/features/`, tagged with the capability and the level (`@cli_lifecycle @e2e`); `@wip` is skipped. The steps drive the built binary named by `SUT_BIN`. Each `.feature` scenario matches a spec scenario by meaning (`~/.claude/rules-custom/gherkin.md`); the design of each change names the test level of every scenario.

## How to verify a change

| When | Command |
|---|---|
| At TDD Red and Green, on the new test | in `tgscout/`: `go test -race -run '<TestName>' ./<package>/...` (add `-tags=integration` for an integration test); in `e2e/`: `SUT_BIN=<built binary> go test -race -count=1 ./...` |
| Once per change, after the last task group | `scripts/qa.sh quick` (vet, lint, architecture, unit tests) or `scripts/qa.sh full` (the default: adds every other check, integration and E2E); `kit verify` in the background runs the level `kit checks-needed` names |
| Before merge | `scripts/qa.sh` and `gitleaks git` |

No check after an edit or a task group beyond the tests `tdd` writes; the checks run once per change.

## Release

The version is the `VERSION` file (SemVer, `~/.claude/rules-custom/semver.md`).

- Every change with behavior change states its bump and target version in the proposal and bumps the version in its last task group; `kit version-check <name>` checks both.
- A change without behavior change states "no release" and leaves the version alone.
- After `kit land`, when `kit release-needed` names a version, run `kit release` (or the project's own release command named here).
