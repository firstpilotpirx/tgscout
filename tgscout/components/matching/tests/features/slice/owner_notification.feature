@owner_notification @slice
Feature: owner-notification — the send of a match through the real matcher and Telegram notifier

  Rule: Send before the processed mark

    Scenario: the message is marked processed only after the send
      Given Telegram has not yet confirmed the send of a match
      When tgscout stops at that moment
      Then the message is not marked processed and the cursor has not moved past it

  Rule: Telegram asks to wait before a send

    Scenario: Telegram asks to wait
      Given Telegram answers a send with a wait of 5 seconds
      When tgscout sends a match
      Then it waits 5 seconds, sends again and marks the message processed after the send is confirmed
