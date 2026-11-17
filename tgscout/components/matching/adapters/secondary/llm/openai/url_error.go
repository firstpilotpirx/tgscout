package openai

import (
	"errors"
	"net/url"
)

// unwrapURL drops the method and address that url.Error repeats.
func unwrapURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
