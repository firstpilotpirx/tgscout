package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLookup(client Client) (*Lookup, *[]time.Duration, *[]time.Duration) {
	var reported, slept []time.Duration
	return &Lookup{
		Client:  client,
		Waiting: func(d time.Duration) { reported = append(reported, d) },
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
	}, &reported, &slept
}

func TestLookupFindsEveryChatAndTheOwner(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined, "-1001": ChatJoined},
		contacts: map[string]bool{"@made_up_owner": true}}
	lookup, _, _ := newTestLookup(client)

	found, err := lookup.Find(context.Background(), []string{"@made_up_market", "-1001"}, "@made_up_owner")

	require.NoError(t, err)
	assert.Equal(t, int64(7), found["@made_up_market"].ID)
	assert.Equal(t, "made_up_market", found["@made_up_market"].Username)
	assert.Equal(t, int64(1), found["-1001"].ID)
}

func TestLookupListsEveryChatNotFound(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined, "@made_up_river": ChatNotJoined},
		contacts: map[string]bool{"@made_up_owner": true}}
	lookup, _, _ := newTestLookup(client)

	_, err := lookup.Find(context.Background(), []string{"@made_up_market", "@made_up_river", "@no_such_chat"},
		"@made_up_owner")

	var notFound *ChatsNotFoundError
	require.True(t, errors.As(err, &notFound))
	assert.Equal(t, []MissingChat{
		{Chat: "@made_up_river", Reason: "the account has not joined it"},
		{Chat: "@no_such_chat", Reason: "no such chat"},
	}, notFound.Chats)
}

func TestLookupSaysWhenTheOwnerIsNotFound(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined}}
	lookup, _, _ := newTestLookup(client)

	_, err := lookup.Find(context.Background(), []string{"@made_up_market"}, "123456789")

	var owner *OwnerNotFoundError
	require.True(t, errors.As(err, &owner))
	assert.Equal(t, "123456789", owner.Owner)
}

func TestLookupWaitsWhenTelegramAsksAndGoesOn(t *testing.T) {
	client := &stubClient{chats: map[string]ChatState{"@made_up_market": ChatJoined},
		contacts: map[string]bool{"@made_up_owner": true},
		floods:   map[string]time.Duration{"@made_up_market": 12 * time.Second}}
	lookup, reported, slept := newTestLookup(client)

	_, err := lookup.Find(context.Background(), []string{"@made_up_market"}, "@made_up_owner")

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{12 * time.Second}, *reported)
	assert.Equal(t, []time.Duration{12 * time.Second}, *slept)
}
