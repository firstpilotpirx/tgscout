//go:build !telegramfake

package telegram

// NewClient builds the MTProto client of the process.
func NewClient(cfg Config) (Client, error) { return newGotdClient(cfg), nil }
