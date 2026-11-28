package telegram

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tgerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapErrorTurnsUnauthorizedIntoRevoked(t *testing.T) {
	for _, kind := range []string{"AUTH_KEY_UNREGISTERED", "SESSION_REVOKED", "USER_DEACTIVATED"} {
		err := mapError(fmt.Errorf("call: %w", tgerr.New(401, kind)))

		assert.True(t, errors.Is(err, ErrSessionRevoked), kind)
	}
}

func TestMapErrorTurnsFloodWaitIntoAWait(t *testing.T) {
	err := mapError(tgerr.New(420, "FLOOD_WAIT_12"))

	var flood *FloodWait
	require.True(t, errors.As(err, &flood))
	assert.Equal(t, 12*time.Second, flood.Wait)
}

func TestMapErrorNamesAWrongCodeAndPassword(t *testing.T) {
	assert.True(t, errors.Is(mapError(tgerr.New(400, "PHONE_CODE_INVALID")), ErrWrongCode))
	assert.True(t, errors.Is(mapError(fmt.Errorf("sign in: %w", auth.ErrPasswordAuthNeeded)), ErrPasswordNeeded))
	assert.True(t, errors.Is(mapError(auth.ErrPasswordInvalid), ErrWrongPassword))
	assert.True(t, errors.Is(mapError(tgerr.New(400, "PASSWORD_HASH_INVALID")), ErrWrongPassword))
}

func TestMapErrorKeepsNoServerText(t *testing.T) {
	err := mapError(errors.New("rpc error: +0000000000 is not allowed"))

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "+0000000000")
	assert.Equal(t, "the call was refused by Telegram", mapError(tgerr.New(400, "PHONE_NUMBER_INVALID")).Error())
}

func TestMapErrorNamesAChatTelegramNoLongerKnows(t *testing.T) {
	for _, kind := range []string{"CHANNEL_INVALID", "CHANNEL_PRIVATE", "CHAT_ID_INVALID", "PEER_ID_INVALID"} {
		err := mapError(fmt.Errorf("get history: %w", tgerr.New(400, kind)))

		assert.True(t, errors.Is(err, ErrChatGone), kind)
	}
}

func TestMapErrorKeepsNilAndCancel(t *testing.T) {
	assert.NoError(t, mapError(nil))
	assert.True(t, errors.Is(mapError(context.Canceled), context.Canceled))
}
