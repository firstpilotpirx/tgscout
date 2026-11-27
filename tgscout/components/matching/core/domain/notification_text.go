package domain

import "fmt"

// notificationTextLimit is how many characters of a message text the owner
// is sent; a longer text ends with «…».
const notificationTextLimit = 1000

// NotificationText is the plain text the owner gets for a match, one part per
// line: the request, the chat, the link when the chat has one and the message
// text. Only the message text is ever cut.
func NotificationText(match Match) string {
	chat := match.Message.ChatTitle
	if chat == "" {
		chat = match.Message.Chat.String()
	}
	head := fmt.Sprintf("Match: %s — %s\nChat: %s\n", match.Request.ID, match.Request.Find, chat)
	if match.Message.Link != "" {
		head += match.Message.Link + "\n"
	}
	return head + cutText(match.Message.Text)
}

func cutText(text string) string {
	runes := []rune(text)
	if len(runes) <= notificationTextLimit {
		return text
	}
	return string(runes[:notificationTextLimit]) + "…"
}
