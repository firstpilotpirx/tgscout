# Tasks

## 1. Domain and settings

- [x] 1.1 Red: unit tests for `domain.P95` (empty gives no value, one sample, nearest rank over 20 samples); `domain.RunStats` (`Cut` gives the last interval's counts and p95 values and starts a new interval, `Total` covers the whole run, a nil `*RunStats` does nothing, latency counted only when given); `domain.LmStudio.Local` (`localhost`, `LOCALHOST`, `127.0.0.1`, `[::1]` local; another host not); the settings parser (`status_interval` read, default 5 minutes, `5s` refused with field and line)
- [x] 1.2 Green: `core/domain/run_stats.go` (with `P95`), `LmStudio.Local`, `Message.Live`, `Settings.StatusInterval`, the YAML parser; `tgscout.yaml.example` names `status_interval`
- [x] 1.3 Commit (`kit ship`)

## 2. Counting in the use cases

- [x] 2.1 Red: unit tests for `CollectMessage` (a live message queued counts `In` and is marked live; an edit, a duplicate and a history message do not count and are not live) and `MatchMessage` (each LLM call recorded, a judged live message records its latency, a history message is judged with no latency, a confirmed send counts a match, a failed judgment counts nothing)
- [x] 2.2 Green: `collect_message.go`, `match_message.go`, `matching.Ports.Stats`, `matching_config.go`
- [x] 2.3 Commit (`kit ship`)

## 3. Console and status loop

- [x] 3.1 Red: unit tests in `apps/cli`: the match line with ` · latency 3.4s` for a live message and none for a history or edited one; the status block (counts with thousands, `≤ 10s` mark, `> 10s` mark, a dash and no mark without samples); the status since start block; the catch-up summary within and over 15 minutes; the non-local warning; the status loop over made-up ticks (a line per tick, the behind warning after 3 growing lines with the three lengths, none after grow-shrink-grow, the next warning only after 3 more growing lines)
- [x] 3.2 Green: `console_status.go` (status block, behind warning, trend, loop), `console_matching.go`, `console_catch_up.go`, `start.go` (non-local warning, status at stop), `live.go` (stats in the ports, status goroutine)
- [x] 3.3 Commit (`kit ship`)

## 4. E2E of the console lines

- [x] 4.1 Red: `e2e/features/quality_targets.feature` (`@quality_targets @e2e`), one `Rule:` per requirement, the E2E scenarios of design D8 but the `@targets` ones: latency on a live match line, none on a catch-up match, the status line with a 10 s interval, status since start at stop, the non-local warning and none for a local address; the config step takes `status_interval` and an LM Studio address
- [x] 4.2 Green: fix what the scenarios find
- [x] 4.3 Commit (`kit ship`)

## 5. Target tests

- [x] 5.1 Red: `@targets` scenarios in `quality_targets.feature` and the `TestTargets` suite (concurrency 1; `TestFeatures` skips `@targets`): the load test (design D9), the catch-up test (design D10), the size test (design D11); the fake Telegram writes the play time of each update and the time of each send, and indexes the history by chat; the fake LM Studio's delay is set per scenario
- [x] 5.2 Green: fix what the tests find; note the time of each target test on this machine in design § Risks
- [x] 5.3 Commit (`kit ship`)

## 6. Docs and version

- [x] 6.1 `README.md` § Targets: the numbers, how to read the status line, `status_interval`, the real-model condition (p95 ≤ 10 s needs a model answering within about 2 s; with a local model the 15-minute catch-up holds for about 2,000 judged messages); `docs/architecture/quality-attributes.html` names the tests that now prove each target
- [x] 6.2 Bump `VERSION` to 0.8.0
- [x] 6.3 Commit (`kit ship`)

Manual checks by the user: none (`manual_run` lists no area); the run with a real LM Studio stays with the owner, before 1.0.0.
