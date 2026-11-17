# quality-targets Specification

## Purpose
TBD - created by archiving change quality-targets. Update Purpose after archive.

## Requirements

### Requirement: Latency on the match line

The console line of a match tgscout sends SHALL show its latency: the time from the message's time in the chat to the moment Telegram confirmed the send to the owner, in seconds with one decimal (`latency 3.4s`). Only a message tgscout got live SHALL show a latency; a match found in a history read (catch-up or a gap read) and a match of an edited message SHALL show none, since their time in the chat is not when tgscout got them.

#### Scenario: latency on a match line

- **GIVEN** a request bound to a watched chat and an LLM that answers yes
- **WHEN** a new message appears in that chat
- **THEN** the console match line shows the latency from the message to the send

#### Scenario: no latency on a catch-up match

- **GIVEN** a matching message posted while tgscout was off
- **WHEN** the owner starts tgscout and it catches up
- **THEN** the console match line of that message shows no latency

### Requirement: Status line

Every `status_interval` while it runs, tgscout SHALL print one status block with: the messages that came in live, the messages judged (live and history), the matches sent, the queue length, the p95 latency and the p95 LLM call time, each over the last interval only. The p95 latency SHALL be marked as meeting the 10 second target or missing it; without a sample, either p95 SHALL show a dash and no mark. The block SHALL never hold a message text.

#### Scenario: the status line

- **GIVEN** tgscout runs with a status interval of 10 seconds and judges live messages
- **WHEN** the status interval passes
- **THEN** the console shows messages in, messages judged, matches, queue length and the p95 latency with its target mark

#### Scenario: an interval without live messages

- **GIVEN** tgscout runs and no message came in during the last interval
- **WHEN** the next status line is printed
- **THEN** it shows zero messages in and a dash for the p95 latency, with no target mark

#### Scenario: a p95 over the target is marked

- **GIVEN** the live messages of the last interval took 12 seconds or more from the chat to the end of their judgment
- **WHEN** the next status line is printed
- **THEN** its p95 latency is marked as over the 10 second target

### Requirement: Latency of a judged message

The latency of a message SHALL run from its time in the chat to the end of its judgment, every match of it sent. Only live messages that are not edits SHALL count in the p95 latency.

#### Scenario: history messages do not count in the p95 latency

- **GIVEN** tgscout judged only messages of a catch-up in the last interval
- **WHEN** the next status line is printed
- **THEN** it shows a dash for the p95 latency

### Requirement: Model behind warning

When the queue lengths of 3 status lines in a row rise each time (such as 97 → 214 → 388), tgscout SHALL print a warning after the third line: the model is behind, the three queue lengths, and a hint to use a faster model or more workers. The warning SHALL come again only after 3 more lines in a row show rising queue lengths.

#### Scenario: the model cannot keep up

- **GIVEN** the queue length grew from one status line to the next, three lines in a row
- **WHEN** the third of those status lines is printed
- **THEN** a warning follows it that the model is behind, with the three queue lengths

#### Scenario: a queue that shrinks once gives no warning

- **GIVEN** the queue length grew, then shrank, then grew in the status lines
- **WHEN** the next status line is printed
- **THEN** no warning that the model is behind is printed

### Requirement: Status at stop

At a stop, after the messages in work are finished and before the stopped line, tgscout SHALL print the status block over the whole run: how long it ran and the same counts, queue length and p95 values, counted since start.

#### Scenario: status since start at stop

- **GIVEN** tgscout ran and judged live messages
- **WHEN** the owner presses Ctrl+C
- **THEN** before the stopped line the console shows the status since start with how long tgscout ran, the counts and the p95 latency with its target mark

### Requirement: Catch-up time against its target

The catch-up summary SHALL mark its time against the 15 minute target: met, or over it with a hint that the model answers too slowly for this many messages.

#### Scenario: catch-up within its target

- **GIVEN** a catch-up that takes less than 15 minutes
- **WHEN** the catch-up summary is printed
- **THEN** its time is marked as within the 15 minute target

#### Scenario: catch-up over its target

- **GIVEN** a catch-up that takes more than 15 minutes
- **WHEN** the catch-up summary is printed
- **THEN** its time is marked as over the 15 minute target, with the hint that the model is too slow for this many messages

### Requirement: Non-local LLM warning

At start, after the config summary, tgscout SHALL print a warning when the host of `lm_studio.url` is not `localhost`, `127.0.0.1` or `::1`: chat texts will leave this machine, with the address and a hint to use localhost. tgscout SHALL keep running after the warning. No message text SHALL go into any output but the console match line.

#### Scenario: a non-local LLM is flagged

- **GIVEN** a config whose LM Studio address points to another machine
- **WHEN** the owner starts tgscout
- **THEN** tgscout warns that chat texts will leave this machine and names the address
- **AND** it keeps running

#### Scenario: a local LLM gives no warning

- **GIVEN** a config whose LM Studio address is on this machine
- **WHEN** the owner starts tgscout
- **THEN** tgscout prints no warning about chat texts leaving the machine

### Requirement: Load target proven

A load test SHALL run tgscout against a made-up Telegram and a made-up LLM that answers each call in 0.5 seconds, with 4 workers and 100 watched chats: a day's flow at its average rate scaled up 20 times, and inside it a peak of 10 messages a second for 10 seconds. Every message SHALL match, so every message ends in a send. The test SHALL pass when the p95 time from a message to its confirmed send is at most 10 seconds.

#### Scenario: peak load meets the speed target

- **GIVEN** 100 watched chats and a peak of 10 messages per second inside a day's flow, with a made-up LLM that answers in half a second
- **WHEN** the load test runs
- **THEN** p95 from message to send is at most 10 seconds

### Requirement: Catch-up target proven

A catch-up test SHALL start tgscout over 100 watched chats with 500 missed messages each, with time scaled 60 times: the 15 minute target becomes 15 seconds. The test SHALL pass when the catch-up summary comes within the scaled 15 minutes of the start and reports 50,000 judged messages.

#### Scenario: catch-up meets its time target

- **GIVEN** 100 chats with 500 missed messages each and a made-up LLM
- **WHEN** tgscout starts
- **THEN** the catch-up summary comes within 15 minutes, scaled

### Requirement: Store size target proven

A size test SHALL fill tgscout's store with 100,000 made-up judged messages, each with one decision, measure the bytes the store takes per judged message and project a year: 5,000 messages a day for 365 days, each judged against 10 requests. The test SHALL pass when the projection is under 10 GB.

#### Scenario: a year of data fits

- **GIVEN** a store filled with a year of decisions at 5,000 messages a day
- **WHEN** its size is measured
- **THEN** it is under 10 GB
