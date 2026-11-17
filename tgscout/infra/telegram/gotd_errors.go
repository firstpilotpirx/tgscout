package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tgerr"
)

// errTelegram stands for any other failure; it keeps no text a server or the
// network sent, which may hold a phone number.
var errTelegram = errors.New("cannot reach Telegram")

// mapError turns a gotd error into an error of the port.
func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, auth.ErrPasswordAuthNeeded):
		return ErrPasswordNeeded
	case errors.Is(err, auth.ErrPasswordInvalid), tgerr.Is(err, "PASSWORD_HASH_INVALID"):
		return ErrWrongPassword
	case tgerr.Is(err, "PHONE_CODE_INVALID", "PHONE_CODE_EMPTY"):
		return ErrWrongCode
	case auth.IsUnauthorized(err):
		return ErrSessionRevoked
	}
	if wait, ok := tgerr.AsFloodWait(err); ok {
		return &FloodWait{Wait: wait}
	}
	if rpc, ok := tgerr.As(err); ok {
		return fmt.Errorf("the call was refused by Telegram: %s", rpc.Type)
	}
	return errTelegram
}
