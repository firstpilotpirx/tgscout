@delivery @e2e
Feature: delivery — no message is lost when LM Studio, Telegram or tgscout itself goes away

  Rule: LLM retries
    # The growing pauses run as a unit test of MatchMessage with a recording sleep.

    Scenario: LM Studio goes away and comes back
      Given LM Studio stops answering and answers yes once it is back
      When a new message appears in a watched chat and LM Studio comes back
      Then the owner gets the match, none lost
      And the console shows one line that LM Studio is not answering and one that it is back

  Rule: A message the model never manages
    # 10 rejected tries and 12 calls with no answer run as unit tests of MatchMessage.

  Rule: LLM status lines
    # One line when it goes and one when it is back, over two messages, runs as a unit
    # test of MatchMessage; the console lines run as unit tests of the console.

  Rule: Send retries
    # A send that fails twice, then works runs in the slice feature of the matcher.

  Rule: Full queue

    Scenario: the queue is full
      Given a queue size of 1, one worker and LM Studio that does not answer
      When three new messages appear and LM Studio comes back after the queue is full
      Then the console shows that the queue is full
      And every one of the three messages is judged and marked processed

  Rule: Telegram reconnect
    # The held cursor while the connection is away runs as unit tests of CatchUp.Pause.

  Rule: Gap read after a reconnect
    # A message processed during the drop runs in the slice feature of catch-up.

    Scenario: the connection to Telegram drops
      Given the LLM answers yes to a request on a watched chat
      When the connection drops, a message is posted during the drop and the connection is back
      Then the message posted during the drop is judged and the owner gets its match
      And the console shows that the connection was lost, that it is back and the gap read

  Rule: Graceful stop

    Scenario: the owner stops tgscout during work
      Given LM Studio answers after 1 second
      When the owner presses Ctrl+C while a message is judged
      Then the console shows the stopping line with 1 message in work
      And tgscout exits with code 0 and the message in work is marked processed

    Scenario: a judgment longer than the grace time
      Given a grace time of 1 second and LM Studio that answers after 5 seconds
      When the owner presses Ctrl+C while a message is judged
      Then tgscout exits with code 0 within 2 seconds
      And the message is not marked processed and the cursor does not move past it

  Rule: End of a stop

    Scenario: a retry pause ends at the stop
      Given LM Studio does not answer
      When the owner presses Ctrl+C while the message waits for its next try
      Then tgscout exits with code 0 within 1 second
      And the message is not marked processed and the cursor does not move past it
      And tgscout says that it stopped

  Rule: Second interrupt

    Scenario: a second Ctrl+C stops at once
      Given LM Studio answers after 5 seconds
      When the owner presses Ctrl+C twice while a message is judged
      Then tgscout ends at once with exit code 130

  Rule: Crash recovery

    Scenario: a crash between send and mark
      Given the LLM answers yes and the confirmation of the send never comes
      When the process is killed after the send and the owner starts tgscout again
      Then the owner gets that match again, and no match is lost

    Scenario: a crash with messages in the queue
      Given one worker and LM Studio that never answers in time
      When three messages arrive, the process is killed and the owner starts tgscout again
      Then every one of those messages is judged after the restart

    Scenario: a clean restart repeats nothing
      Given every message read so far is processed after a match
      When the owner stops and starts tgscout
      Then no match is sent again and the LLM is not asked again
