package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

type fakeSettingsRepository struct{ settings domain.Settings }

func (f fakeSettingsRepository) Settings() domain.Settings { return f.settings }

func TestSettingsRepositoryGivesTheResolvedSettings(t *testing.T) {
	var port application.SettingsRepository = fakeSettingsRepository{settings: domain.Settings{Workers: 2}}

	assert.Equal(t, 2, port.Settings().Workers)
}
