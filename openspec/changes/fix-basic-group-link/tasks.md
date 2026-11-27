# Tasks

## 1. Basic group: no link line

Files: `tgscout/infra/telegram/`, `tgscout/components/matching/adapters/primary/collect/telegram/`, `tgscout/components/matching/core/domain/notification_text*.go`, `tgscout/apps/cli/live.go`
Needs: none

- [ ] 1.1 Red: unit tests: the collector gives an empty link for a basic group, keeps `t.me/c/<id>/<msg>` for a supergroup without a username ("a chat without a username") and `t.me/<username>/<msg>` for a public chat; `NotificationText` leaves out the link line when the link is empty and keeps it, whole, with a long text ("a long message text is cut", unchanged); the fake client marks `-<id>` (not `-100<id>`) as a basic group
- [ ] 1.2 Green: `ChatInfo.Basic` set from the Telegram type in the gotd client (dialog peer `InputPeerChat`, resolved `tg.Chat`) and from the reference in the fake client; `collect.Chat.Basic` filled in `live.go`; the collector's link empty for a basic group; `NotificationText` drops an empty link line
- [ ] 1.3 Commit the group (`kit ship`)

## 2. E2E of a basic group match

Files: `e2e/features/owner_notification.feature`, `e2e/` steps
Needs: 1

- [ ] 2.1 Red: scenario "a basic group has no link line" under `Rule: Link to the message` in `owner_notification.feature`, with a made-up basic group ID; the rule's comment still names the unit test of "a chat without a username"
- [ ] 2.2 Green: the steps it needs; fix what it finds
- [ ] 2.3 Commit the group (`kit ship`)

## 3. Release

Files: `VERSION`
Needs: 1, 2

- [ ] 3.1 Bump `VERSION` to 0.8.1
- [ ] 3.2 Commit the group (`kit ship`)
