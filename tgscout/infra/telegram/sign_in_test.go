package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignInFromASavedSessionAsksNothing(t *testing.T) {
	client := &stubClient{authorized: true}
	prompter := &scriptedPrompter{}

	account, err := SignIn(context.Background(), client, prompter, &stubSession{})

	require.NoError(t, err)
	assert.Equal(t, "Made Up Scout", account.Name)
	assert.Empty(t, prompter.notices)
	assert.Empty(t, client.calls)
}

func TestSignInFirstRunAsksPhoneCodeAndPassword(t *testing.T) {
	client := &stubClient{code: "24680", password: "made-up-pass"}
	prompter := &scriptedPrompter{phones: []string{"+0000000000"}, codes: []string{"24680"},
		passwords: []string{"made-up-pass"}}
	session := &stubSession{firstRun: true}

	account, err := SignIn(context.Background(), client, prompter, session)

	require.NoError(t, err)
	assert.Equal(t, "Made Up Scout", account.Name)
	assert.True(t, client.signedIn)
	assert.True(t, session.committed)
	assert.Equal(t, []Notice{NoticeFirstSignIn, NoticeCodeSent}, prompter.notices)
}

func TestSignInAsksAgainOnAWrongCodeAndPassword(t *testing.T) {
	client := &stubClient{code: "24680", password: "made-up-pass"}
	prompter := &scriptedPrompter{phones: []string{"+0000000000"}, codes: []string{"11111", "24680"},
		passwords: []string{"wrong", "made-up-pass"}}
	session := &stubSession{firstRun: true}

	_, err := SignIn(context.Background(), client, prompter, session)

	require.NoError(t, err)
	assert.Equal(t, []Notice{NoticeFirstSignIn, NoticeCodeSent, NoticeWrongCode, NoticeWrongPassword},
		prompter.notices)
	assert.True(t, session.committed)
}

func TestSignInCommitsNothingWhenTheOwnerGivesUp(t *testing.T) {
	client := &stubClient{code: "24680"}
	prompter := &scriptedPrompter{phones: []string{"+0000000000"}, codes: []string{"11111"}}
	session := &stubSession{firstRun: true}

	_, err := SignIn(context.Background(), client, prompter, session)

	require.Error(t, err)
	assert.False(t, session.committed)
}

func TestSignInWithARejectedSavedSessionIsRevoked(t *testing.T) {
	for _, client := range []*stubClient{{authorized: false}, {authErr: ErrSessionRevoked}} {
		_, err := SignIn(context.Background(), client, &scriptedPrompter{}, &stubSession{})

		assert.True(t, errors.Is(err, ErrSessionRevoked))
	}
}
