## Why

This is the heart of tgscout: a new message in a watched chat is checked by the local LLM against every standing request of that chat, and a match is shown at once. Only the LLM compares a message with a request — no text search, no embeddings in version 1. This slice runs the whole message path from Telegram to the console: collector, in-memory queue, matcher workers, the llm part that alone talks to LM Studio, the decisions and the cursor in SQLite. The private message to the owner follows in the next slice; here the console is the notifier, which the design keeps anyway for debugging.

Design pages this slice implements:

- [Message path](../../../docs/flows/message-path.html) — the sequence and the activity: processed already? → queue with the requests of its chat → ask the llm per request → save the decision → match? → send → only then mark processed and move the cursor.
- [L3 Component — tgscout](../../../docs/architecture/tgscout.html) — collector, queue (Go channel), matcher (Go workers), llm, notifier, repository; the queue message picture (`chat_id`, `message_id`, `sent_at`, `text`, `link`, the requests of its chat).
- [L4 Code — matching](../../../docs/modules/matching/overview.html) — use cases `CollectMessage` and `MatchMessage`; ports `MessageRepository`, `MessageQueue`, `Llm`, `Notifier`; adapters `primary/collect/telegram`, `primary/match/channel`, `ChannelMessageQueue`, `OpenAiLlmClient`, `SqliteMessageRepository`.
- [store — SQLite schema](../../../docs/architecture/store.html) — `decision` per message and request, `processed_message` written last, `chat_cursor` moved after it.
- [L1 System Context § About the system](../../../docs/architecture/context.html) — the pre-filter drops only service, empty and duplicate messages; a photo's caption counts as text.

## What Changes

- The collector listens to new messages of the watched chats over the shared MTProto client and calls `CollectMessage`.
- `CollectMessage` drops service messages (joins, pins, title changes), messages with no text, and a message it already saw; a photo or a file with a caption is judged by its caption. It asks the repository whether the message was processed already; if not, it puts one queue item into the queue: chat, message ID, time, text, a link to the message, and the requests of its chat (so the matcher never reads the config).
- The queue is a buffered Go channel (`ChannelMessageQueue`); `workers` matcher workers take items from it.
- `MatchMessage` asks the `Llm` port once per request of the item, saves each decision (`matched`, `decided_at`), and hands every «yes» to the `Notifier` port; after the last request, and after every «yes» was confirmed sent, it marks the message processed and moves the cursor.
- The cursor of a chat moves only past messages that are all processed: with several workers finishing out of order, it never passes an unprocessed message.
- `OpenAiLlmClient` calls the chat-completions endpoint of the OpenAI-compatible API at `lm_studio.url` with `lm_studio.model` and `lm_studio.timeout`; the prompt holds the request text and the message text and asks for a strict yes or no; the client turns the answer into a decision.
- An LLM call that fails (LM Studio down, timeout) leaves the message unprocessed and prints an error line; the full retry policy is `delivery-resilience`.
- The notifier of this slice is the console: one line per match with time, request id, chat, link and the start of the text.
- Each «yes» decision row is a found message; nothing else stores matches (no matches file).
- An edited message is judged again only for the requests with no «yes» shown yet; a «yes» saved but not shown is judged and shown again; a shown «yes» is never shown twice for the same request.
- An LLM answer that is neither yes nor no is asked once more with a stricter prompt; a second unclear answer is saved as «no» and a warning line with the raw answer is printed.
- The prompt keeps the fixed matching rules in the system message and puts the request text and the message text in the user message.
- A new root setting `queue_size` (default 1,000, at least 1) sets the queue capacity; when the queue is full, the collector waits (back-pressure).
- A duplicate is the same chat and message ID seen twice; the same text in another message is judged again.

Current stage: PoC. Target stage: PoC.
Scope: collector for new messages, the pre-filter, the processed check, the queue, matcher workers, the llm part over the OpenAI-compatible API, decisions, processed marks, the cursor, the console notifier, re-judging edited messages, the `queue_size` setting.
Acceptance bar: the scenarios below pass at the levels design § Test levels names (E2E with a fake Telegram update stream and a fake OpenAI-compatible server, slice, unit); a manual run with LM Studio prints a match on the console for a made-up message posted in a test chat.
Test override: the E2E scenarios talk to a made-up OpenAI-compatible server (`httptest`) instead of LM Studio, overriding `testing-e2e-go.md` «External deps: Real or production-equivalent. No httptest» for the LLM only; decided by the prepared run on the owner's order of 2026-11-17 (decide and finish).
Out of scope: the private Telegram message to the owner (`owner-notification`); history at start (`catch-up`); retries, backoff, graceful drain and reconnects (`delivery-resilience`); latency and load measurement (`quality-targets`); images, video, audio; owner feedback and commands (later, not now).
Interface: CLI — match lines and error lines on the console.

## Acceptance scenarios

- Scenario: a matching message is shown on the console
  - Given a request "a kids' bike, 16 inch, under 100 €" bound to a watched chat, and the LLM answers yes
  - When a new message "Selling a 16-inch kids' bike, 80 €" appears in that chat
  - Then the console shows a match line with the request, the chat and a link to the message
  - And the decision is saved as a match, the message is marked processed and the cursor moves to it
- Scenario: a message that does not match
  - Given the LLM answers no
  - When a new message appears in a watched chat
  - Then the console shows no match line, the decision is saved as no match and the message is marked processed
- Scenario: one message, two requests of its chat
  - Given two requests bound to the same chat
  - When a new message appears in that chat
  - Then the LLM is asked once per request and two decisions are saved
- Scenario: a service message is dropped
  - Given a watched chat
  - When someone joins the chat
  - Then the LLM is not asked and nothing is saved
- Scenario: a message with no text is dropped
  - Given a watched chat
  - When a photo without a caption appears in it
  - Then the LLM is not asked
- Scenario: a photo caption counts as text
  - Given a watched chat
  - When a photo with a caption appears in it
  - Then the LLM is asked about the caption
- Scenario: a processed message is skipped
  - Given a message that is already marked processed
  - When the same message arrives again
  - Then the LLM is not asked again and no second match line appears
- Scenario: LM Studio is down
  - Given LM Studio does not answer
  - When a new message appears in a watched chat
  - Then the console shows an error line, the message is not marked processed and the cursor does not move past it
- Scenario: the cursor never passes an unprocessed message
  - Given messages 10, 11 and 12 of one chat in the queue and two workers
  - When message 12 is processed before message 11
  - Then the cursor of the chat stays at 10 until message 11 is processed, then moves to 12
- Scenario: an edited message is judged again only where it was «no»
  - Given a processed message with a «yes» for one request and a «no» for another
  - When the message is edited
  - Then the LLM is asked only about the request with «no», and no second match line appears for the other

## Capabilities

### New Capabilities

- `message-matching`: from a new message in a watched chat to a saved decision per request and a shown match, with the processed mark and the cursor.

### Modified Capabilities

- `settings`: the new root setting `queue_size` (default 1000, at least 1).

## Impact

- New: `adapters/secondary/notify/console`, `adapters/primary/collect/telegram`, `adapters/primary/match/channel`, `adapters/secondary/queue/channel`, `adapters/secondary/llm/openai`, `adapters/secondary/persistence/sqlite` (repository methods), the console notifier, domain `Message`, `Decision`, `Cursor`, `MessageTracker`, use cases `CollectMessage`, `MatchMessage`.
- Changed: `apps/cli/main.go` (start the workers and the listener, as the start flow says).

## Open points

None: the points raised earlier are decided in What Changes.

## Release

SemVer: MINOR — tgscout finds matches in new messages.
Target version: 0.4.0.
