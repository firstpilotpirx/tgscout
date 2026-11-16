package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

type fakeNotifier struct{}

func TestNotifierAcceptsAnAdapter(t *testing.T) {
	var port application.Notifier = fakeNotifier{}
	assert.NotNil(t, port)
}
