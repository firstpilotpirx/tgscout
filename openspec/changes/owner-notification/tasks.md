## 1. Text of the private message

- [x] 1.1 Red: unit tests for `domain.NotificationText` — the four lines; the title falls back to the config text; a 1,500-character text cut to 1,000 and `…` with the request and the link whole; a text of exactly 1,000 kept; line breaks kept
- [x] 1.2 Green: `core/domain/notification_text.go`, `Message.ChatTitle`
- [x] 1.3 Commit (`kit ship`)

## 2. Telegram client

- [x] 2.1 Red: unit tests for `telegram.Retry` (a FLOOD_WAIT reported, slept and retried; another error returned; a stop during the sleep returned), `Lookup.Find` with `OwnerMissing`, `OwnerNoChat` (`OwnerChatClosedError`) and `OwnerReady`, `ChatInfo.Title` from the fake, the fake's `SendToOwner` (writes `sent_file` lines, `send_fails`, `send_flood_wait_seconds` once) and its `owner_chat_closed`, `private` updates dropped; the gotd `mapMessage` test for a `PeerUser` message; update the stub client
- [x] 2.2 Green: `infra/telegram` `OwnerState`, `FindOwner`, `SendToOwner`, `ChatInfo.Title`, `Retry`, `OwnerChatClosedError`; the gotd client (kept owner peer, `messages.getPeerDialogs`, `messages.sendMessage`, titles from resolved chats and dialog entities); the fake client
- [x] 2.3 Commit (`kit ship`)

## 3. Notifier adapter and collector

- [x] 3.1 Red: integration tests for `TelegramNotifierClient` — the text sent to the owner, the printer called after the send with the same text, a FLOOD_WAIT retried after the reported wait, a send error returned with no print, a printer error dropped after the send; unit test for the collector putting the chat title into the message
- [x] 3.2 Green: `adapters/secondary/notify/telegram/telegram_notifier_client.go`; the collector's `Chat.Title`
- [x] 3.3 Red: `components/matching/tests/features/slice/owner_notification.feature` (`@owner_notification @slice`): the message is marked processed only after the send; Telegram asks to wait — through the real `MatchMessage` and `TelegramNotifierClient` with a stub client and the memory repository; a `MatchMessage` unit test that a `Notify` stopped by the context leaves the message unprocessed and the cursor in place
- [x] 3.4 Green: what the slice scenarios find; `StageNotify` prints as `send`
- [x] 3.5 Commit (`kit ship`)

## 4. CLI

- [ ] 4.1 Red: unit tests for the console lines of design § 6 (match sent with the indented text, FLOOD_WAIT before a send, send failed, cannot write to the owner) and for `reportLookup` with `OwnerChatClosedError`; update the start tests for the wired Telegram notifier
- [ ] 4.2 Green: `apps/cli/console_matching.go`, `console_telegram.go`, `connect.go`, `live.go` (Telegram notifier over the shared client, the owner, titles of the found chats); remove `adapters/secondary/notify/console`
- [ ] 4.3 Commit (`kit ship`)

## 5. E2E

- [ ] 5.1 Red: `e2e/features/owner_notification.feature` (`@owner_notification @e2e`), one `Rule:` per requirement, every E2E scenario of design § Test levels; the world's fake Telegram gets `sent_file`, `send_fails`, `titles`, `owner_chat_closed` and `private` updates; the owner-chat row in `e2e/features/telegram_session.feature`; the match line steps of `message_matching.feature` follow the new line
- [ ] 5.2 Green: fix what the scenarios find
- [ ] 5.3 Commit (`kit ship`)

## 6. Docs and version

- [ ] 6.1 `README.md`: Setup → First contact (write «hi» to tgscout's account once, add it to contacts), what the owner gets in Telegram, the new console lines
- [ ] 6.2 Bump `VERSION` to 0.5.0
- [ ] 6.3 Commit (`kit ship`)

Manual checks by the user: none for this run (`manual_run` lists no area); the owner's run with real accounts of the acceptance bar stays for the owner.
