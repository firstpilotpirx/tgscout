package domain

import "fmt"

// notificationTextLimit is how many characters of a message text the owner
// is sent; a longer text ends with «…».
const notificationTextLimit = 1000

// NotificationText is the plain text the owner gets for a match, one part per
// line: the request, the chat, the link and the message text. Only the
// message text is ever cut.
func NotificationText(match Match) string {
	chat := match.Message.ChatTitle
	if chat == "" {
		chat = match.Message.Chat.String()
	}
	return fmt.Sprintf("Match: %s — %s\nChat: %s\n%s\n%s", match.Request.ID, match.Request.Find,
		chat, match.Message.Link, cutText(match.Message.Text))
}

func cutText(text string) string {
	runes := []rune(text)
	if len(runes) <= notificationTextLimit {
		return text
	}
	return string(runes[:notificationTextLimit]) + "…"
}
