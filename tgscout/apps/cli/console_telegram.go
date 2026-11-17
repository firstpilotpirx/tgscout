package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// The console lines of the sign-in and the lookup (the approved mock).

func (c *console) firstSignIn() {
	c.println(c.out, c.cyan.Sprint(c.marks.run)+" no session yet — first sign-in")
}

func (c *console) codeSent() { c.println(c.out, "Telegram sent a login code to the account's app.") }

func (c *console) wrongCode() {
	c.println(c.errOut, c.red.Sprint(c.marks.fail)+" the code is wrong — try again")
}

func (c *console) wrongPassword() {
	c.println(c.errOut, c.red.Sprint(c.marks.fail)+" the password is wrong — try again")
}

func (c *console) signedIn(name string, saved bool) {
	text := "signed in as " + name
	if saved {
		text += " (saved session)"
	}
	c.done(text)
}

func (c *console) sessionSaved(path string) {
	c.done(fmt.Sprintf("session saved %s (mode 600)", c.cyan.Sprint(path)))
}

func (c *console) floodWait(wait time.Duration) {
	c.println(c.errOut, fmt.Sprintf("%s Telegram asks to wait %s (FLOOD_WAIT), then goes on",
		c.yellow.Sprint(c.marks.warn), wait))
}

// sendFloodWait is the warning of a FLOOD_WAIT before a match is sent again.
func (c *console) sendFloodWait(wait time.Duration) {
	c.println(c.errOut, fmt.Sprintf("%s Telegram asks to wait %s (FLOOD_WAIT), then sends again",
		c.yellow.Sprint(c.marks.warn), wait))
}

func (c *console) ownerFound(owner string) { c.done("owner found: " + owner) }

func (c *console) watching(chats int) { c.done("watching " + count(chats, "chat")) }

// chatsNotFound prints every chat the account cannot watch, aligned, and the fix.
func (c *console) chatsNotFound(chats []telegram.MissingChat) {
	c.println(c.errOut, fmt.Sprintf("%s %s not found", c.red.Sprint(c.marks.fail), count(len(chats), "chat")))
	width := 0
	for _, chat := range chats {
		width = max(width, len(chat.Chat))
	}
	for _, chat := range chats {
		name := chat.Chat + strings.Repeat(" ", width-len(chat.Chat))
		c.println(c.errOut, fmt.Sprintf("  %s  %s", c.red.Sprint(name), chat.Reason))
	}
	c.println(c.errOut, "  "+c.dim.Sprint("join them with tgscout's account, or remove them from tgscout.yaml"))
}
