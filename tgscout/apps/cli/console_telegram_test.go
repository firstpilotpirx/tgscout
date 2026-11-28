package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

func TestConsoleSignInLines(t *testing.T) {
	c, out, errOut := newTestConsole(true)

	c.firstSignIn()
	c.codeSent()
	c.signedIn("Made Up Scout", false)
	c.sessionSaved("./tgscout.session")
	c.signedIn("Made Up Scout", true)
	c.ownerFound("@made_up_owner")
	c.watching(5, telegram.LookupCounts{})
	c.watching(1, telegram.LookupCounts{})
	c.wrongCode()
	c.wrongPassword()
	c.floodWait(12 * time.Second)
	c.sendFloodWait(5 * time.Second)

	assert.Equal(t, "[run] no session yet — first sign-in\n"+
		"Telegram sent a login code to the account's app.\n"+
		"[ok] signed in as Made Up Scout\n"+
		"[ok] session saved ./tgscout.session (mode 600)\n"+
		"[ok] signed in as Made Up Scout (saved session)\n"+
		"[ok] owner found: @made_up_owner\n"+
		"[ok] watching 5 chats\n"+
		"[ok] watching 1 chat\n", out.String())
	assert.Equal(t, "[fail] the code is wrong — try again\n"+
		"[fail] the password is wrong — try again\n"+
		"[warn] Telegram asks to wait 12s (FLOOD_WAIT), then goes on\n"+
		"[warn] Telegram asks to wait 5s (FLOOD_WAIT), then sends again\n", errOut.String())
}

func TestConsoleWatchingNamesTheLookupCounts(t *testing.T) {
	c, out, _ := newTestConsole(true)

	c.watching(3, telegram.LookupCounts{LookedUp: 3})
	c.watching(4, telegram.LookupCounts{Stored: 3, LookedUp: 1})
	c.watching(3, telegram.LookupCounts{Stored: 3})

	assert.Equal(t, "[ok] watching 3 chats · 3 looked up\n"+
		"[ok] watching 4 chats · 3 from the store, 1 looked up\n"+
		"[ok] watching 3 chats · 3 from the store\n", out.String())
}

func TestConsoleChatsNotFoundListsEveryChat(t *testing.T) {
	c, _, errOut := newTestConsole(true)

	c.chatsNotFound([]telegram.MissingChat{
		{Chat: "@made_up_river", Reason: "the account has not joined it"},
		{Chat: "@no_such_chat", Reason: "no such chat"},
	})

	assert.Equal(t, "[fail] 2 chats not found\n"+
		"  @made_up_river  the account has not joined it\n"+
		"  @no_such_chat   no such chat\n"+
		"  join them with tgscout's account, or remove them from tgscout.yaml\n", errOut.String())
}
