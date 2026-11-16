package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

type fakeMessageRepository struct{}

func TestMessageRepositoryAcceptsAnAdapter(t *testing.T) {
	var port application.MessageRepository = fakeMessageRepository{}
	assert.NotNil(t, port)
}
