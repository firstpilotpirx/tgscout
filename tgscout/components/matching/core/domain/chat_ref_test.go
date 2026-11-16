package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func TestNewChatRefAcceptsEveryForm(t *testing.T) {
	for _, text := range []string{
		"@made_up_market",
		"https://t.me/made_up_market",
		"t.me/made_up_market",
		"1234567",
		"-1001234567890",
	} {
		ref, err := domain.NewChatRef(text)

		require.NoError(t, err, text)
		assert.Equal(t, text, ref.String())
	}
}

func TestNewChatRefRejectsMalformedText(t *testing.T) {
	for _, text := range []string{
		"", "  ", "@abc", "@1abcde", "made_up_market", "https://example.com/made_up_market",
		"@made up", "12a4", "-", "https://t.me/",
	} {
		_, err := domain.NewChatRef(text)

		assert.ErrorIs(t, err, domain.ErrBadChatRef, text)
	}
}

func TestChatRefKeyJoinsLinkAndUsername(t *testing.T) {
	at, _ := domain.NewChatRef("@Made_Up_Market")
	link, _ := domain.NewChatRef("https://t.me/made_up_market")
	id, _ := domain.NewChatRef("-1001234567890")

	assert.Equal(t, at.Key(), link.Key())
	assert.Equal(t, "-1001234567890", id.Key())
}

func TestNewOwnerAcceptsUsernameAndID(t *testing.T) {
	for _, text := range []string{"@made_up_owner", "1234567"} {
		owner, err := domain.NewOwner(text)

		require.NoError(t, err, text)
		assert.Equal(t, text, owner.String())
	}
}

func TestNewOwnerRejectsLinkAndMalformedText(t *testing.T) {
	for _, text := range []string{"", "https://t.me/made_up_owner", "t.me/made_up_owner", "owner", "@ab"} {
		_, err := domain.NewOwner(text)

		assert.ErrorIs(t, err, domain.ErrBadOwner, text)
	}
}
