package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

func TestNewMatchMessageBuildsTheUseCase(t *testing.T) {
	useCase := application.NewMatchMessage(fakeMessageRepository{}, fakeLlm{}, fakeNotifier{})
	assert.NotNil(t, useCase)
}
