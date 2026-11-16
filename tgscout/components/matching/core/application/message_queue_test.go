package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

type fakeMessageQueue struct{}

func TestMessageQueueAcceptsAnAdapter(t *testing.T) {
	var port application.MessageQueue = fakeMessageQueue{}
	assert.NotNil(t, port)
}
