@quality_targets @e2e
Feature: quality-targets — the version 1 targets shown on the console and proven by tests

  Rule: Latency on the match line

    Scenario: latency on a match line
      Given a request bound to a watched chat and the LLM answers yes
      When a new message appears in that chat
      Then the console match line shows the latency from the message to the send

    Scenario: no latency on a catch-up match
      Given a matching message posted while tgscout was off
      When the owner starts tgscout and it catches up
      Then the console match line of that message shows no latency

  Rule: Status line
    # An interval without live messages and a p95 over the target run as unit
    # tests of the console (design D8).

    Scenario: the status line
      Given tgscout runs with a status interval of 10 seconds and judges live messages
      When the status interval passes
      Then the console shows messages in, messages judged, matches, queue length and the p95 latency with its target mark

  Rule: Latency of a judged message
    # History messages left out of the p95 run as a unit test of the match use case.

  Rule: Model behind warning
    # The warning after 3 growing lines, and none after a queue that shrank,
    # run as unit tests of the status loop with made-up ticks (design D8).

  Rule: Status at stop

    Scenario: status since start at stop
      Given tgscout ran and judged live messages
      When the owner presses Ctrl+C
      Then before the stopped line the console shows the status since start with how long tgscout ran, the counts and the p95 latency with its target mark

  Rule: Catch-up time against its target
    # Within and over 15 minutes run as unit tests of the console summary.

  Rule: Non-local LLM warning

    Scenario: a non-local LLM is flagged
      Given a config whose LM Studio address points to another machine
      When the owner starts tgscout
      Then tgscout warns that chat texts will leave this machine and names the address
      And it keeps running

    Scenario: a local LLM gives no warning
      Given a config whose LM Studio address is on this machine
      When the owner starts tgscout
      Then tgscout prints no warning about chat texts leaving the machine
