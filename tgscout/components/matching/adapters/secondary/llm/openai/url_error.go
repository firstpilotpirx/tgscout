package openai

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

// callError is a failed call to LM Studio: its text names only the address
// (and the timeout); the cause stays for errors.Is.
type callError struct {
	text  string
	cause error
}

func (e *callError) Error() string { return e.text }

func (e *callError) Unwrap() error { return e.cause }

// reachError tells why the address gave no answer: the timeout, or no answer at all.
func reachError(base string, timeout time.Duration, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if urlErr != nil && urlErr.Timeout() {
		return &callError{text: fmt.Sprintf("%s did not answer within %s", base, timeout), cause: err}
	}
	return &callError{text: "cannot reach " + base, cause: err}
}

// rejectedError is an answer about the message itself: a client error status
// or an answer that cannot be read. It is application.ErrLlmRejected for errors.Is.
type rejectedError struct {
	text  string
	cause error
}

func (e *rejectedError) Error() string { return e.text }

func (e *rejectedError) Unwrap() error { return e.cause }

func (e *rejectedError) Is(target error) bool { return target == application.ErrLlmRejected }
