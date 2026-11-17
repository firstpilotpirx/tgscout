package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func bikeMatch(t *testing.T, title, text string) domain.Match {
	t.Helper()
	ref, err := domain.NewChatRef("@made_up_market")
	require.NoError(t, err)
	return domain.Match{
		Request: domain.Request{ID: "kids-bike", Find: "a kids' bike, 16 inch, under 100 €"},
		Message: domain.Message{Chat: ref, ChatTitle: title, Text: text, Link: "https://t.me/made_up_market/4211"},
	}
}

func TestNotificationTextHasFourParts(t *testing.T) {
	text := domain.NotificationText(bikeMatch(t, "Made-up market", "Selling a 16-inch kids' bike, 80 €"))

	assert.Equal(t, "Match: kids-bike — a kids' bike, 16 inch, under 100 €\n"+
		"Chat: Made-up market\n"+
		"https://t.me/made_up_market/4211\n"+
		"Selling a 16-inch kids' bike, 80 €", text)
}

func TestNotificationTextNamesTheChatByConfigTextWithoutTitle(t *testing.T) {
	text := domain.NotificationText(bikeMatch(t, "", "bike"))

	assert.Contains(t, text, "\nChat: @made_up_market\n")
}

func TestNotificationTextCutsALongText(t *testing.T) {
	long := strings.Repeat("é", 1500)

	text := domain.NotificationText(bikeMatch(t, "Made-up market", long))

	lines := strings.SplitN(text, "\n", 4)
	require.Len(t, lines, 4)
	assert.Equal(t, "Match: kids-bike — a kids' bike, 16 inch, under 100 €", lines[0])
	assert.Equal(t, "https://t.me/made_up_market/4211", lines[2])
	assert.Equal(t, strings.Repeat("é", 1000)+"…", lines[3])
}

func TestNotificationTextKeepsATextOfTheLimit(t *testing.T) {
	exact := strings.Repeat("a", 1000)

	text := domain.NotificationText(bikeMatch(t, "Made-up market", exact))

	assert.True(t, strings.HasSuffix(text, "\n"+exact), "the text is kept whole")
}

func TestNotificationTextKeepsLineBreaks(t *testing.T) {
	text := domain.NotificationText(bikeMatch(t, "Made-up market", "Kids' bike\n16 inch\n80 €"))

	assert.True(t, strings.HasSuffix(text, "\nKids' bike\n16 inch\n80 €"))
}
