package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func madeUpMessage(t *testing.T, chat string, id int, text string) domain.Message {
	t.Helper()
	ref, err := domain.NewChatRef(chat)
	require.NoError(t, err)
	return domain.Message{
		Key:    domain.MessageKey{ChatID: 7001, MessageID: id},
		Chat:   ref,
		SentAt: time.Date(2026, 11, 17, 14, 2, 0, 0, time.Local),
		Text:   text,
		Link:   "https://t.me/made_up_market/" + strings.TrimPrefix(chat, "@"),
	}
}

func TestPrintMatchShowsTimeRequestChatLinkAndText(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	message := madeUpMessage(t, "@made_up_market", 4211, "Selling a 16-inch kids' bike, 80 €")
	message.Link = "https://t.me/made_up_market/4211"

	err := c.PrintMatch(domain.Match{Message: message, Request: domain.Request{ID: "kids-bike"}})

	require.NoError(t, err)
	assert.Equal(t,
		"[ok] match  14:02  kids-bike  @made_up_market  https://t.me/made_up_market/4211\n"+
			"         Selling a 16-inch kids' bike, 80 €\n",
		out.String())
	assert.Empty(t, errOut.String())
}

func TestPrintMatchCutsTheTextAt80CharactersAndJoinsLines(t *testing.T) {
	c, out, _ := newTestConsole(true)
	text := "Selling a 16-inch kids' bike,\n80 €, pick-up in the old town, " + strings.Repeat("ä", 60)
	message := madeUpMessage(t, "@made_up_market", 4211, text)

	require.NoError(t, c.PrintMatch(domain.Match{Message: message, Request: domain.Request{ID: "kids-bike"}}))

	second := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")[1]
	start := strings.TrimPrefix(second, "         ")
	assert.Equal(t, 81, len([]rune(start)), "80 characters and the cut mark")
	assert.True(t, strings.HasSuffix(start, "…"))
	assert.True(t, strings.HasPrefix(start, "Selling a 16-inch kids' bike, 80 €"), "a line break becomes a space")
}

func TestPrintMatchWithSymbols(t *testing.T) {
	c, out, _ := newTestConsole(false)
	message := madeUpMessage(t, "@made_up_market", 4211, "Desk lamp, clamp mount, 10 €")

	require.NoError(t, c.PrintMatch(domain.Match{Message: message, Request: domain.Request{ID: "desk-lamp"}}))

	assert.Contains(t, out.String(), "✓")
	assert.Contains(t, out.String(), " match  14:02  ")
	assert.Contains(t, out.String(), "\n         Desk lamp, clamp mount, 10 €\n")
}

func TestUnclearLineShowsTheRawAnswerCut(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	item := domain.QueueItem{Message: madeUpMessage(t, "@made_up_market", 4215, "secret message text")}

	c.Unclear(item, application.Unclear{Request: domain.Request{ID: "kids-bike", Find: "secret request"},
		Raw: "It might be, the size is not stated"})
	c.Unclear(item, application.Unclear{Request: domain.Request{ID: "kids-bike"}, Raw: strings.Repeat("x", 100)})

	lines := strings.Split(errOut.String(), "\n")
	assert.Equal(t, "[warn] unclear LLM answer for kids-bike on @made_up_market/4215, saved as no", lines[0])
	assert.Equal(t, `  answer: "It might be, the size is not stated"`, lines[1])
	assert.Equal(t, `  answer: "`+strings.Repeat("x", 80)+`…"`, lines[3])
	assert.NotContains(t, errOut.String(), "secret")
	assert.Empty(t, out.String())
}

func TestFailedLineNamesChatAndMessageNeverTheTexts(t *testing.T) {
	c, out, errOut := newTestConsole(true)
	item := domain.QueueItem{
		Message:  madeUpMessage(t, "@made_up_market", 4216, "secret message text"),
		Requests: []domain.Request{{ID: "kids-bike", Find: "secret request"}},
	}

	c.Failed(item, &application.MatchError{Key: item.Message.Key, Stage: application.StageLlm,
		Err: errors.New("cannot reach http://localhost:1234/v1")})

	assert.Equal(t,
		"[fail] LLM failed on @made_up_market/4216: cannot reach http://localhost:1234/v1\n"+
			"  the message stays unprocessed; is LM Studio running?\n",
		errOut.String())
	assert.Empty(t, out.String())
}

func TestFailedLineOfAnotherStage(t *testing.T) {
	c, _, errOut := newTestConsole(true)
	item := domain.QueueItem{Message: madeUpMessage(t, "@made_up_market", 4217, "secret")}

	c.Failed(item, &application.MatchError{Key: item.Message.Key, Stage: application.StageStore,
		Err: errors.New("disk I/O error")})

	assert.Equal(t,
		"[fail] database failed on @made_up_market/4217: disk I/O error\n"+
			"  the message stays unprocessed\n",
		errOut.String())
}

func TestCollectFailedLine(t *testing.T) {
	c, _, errOut := newTestConsole(true)

	c.collectFailed(madeUpMessage(t, "@made_up_market", 4218, "secret"), errors.New("disk I/O error"))

	assert.Equal(t,
		"[fail] cannot check @made_up_market/4218: disk I/O error\n"+
			"  the message stays unprocessed\n",
		errOut.String())
}
