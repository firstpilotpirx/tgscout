package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
)

func TestNewCollectMessageBuildsTheUseCase(t *testing.T) {
	useCase := application.NewCollectMessage(fakeSettingsRepository{}, fakeMessageRepository{}, fakeMessageQueue{})
	assert.NotNil(t, useCase)
}
