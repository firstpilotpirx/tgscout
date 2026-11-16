package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

type fakeLlm struct{}

func TestLlmAcceptsAnAdapter(t *testing.T) {
	var port application.Llm = fakeLlm{}
	assert.NotNil(t, port)
}
