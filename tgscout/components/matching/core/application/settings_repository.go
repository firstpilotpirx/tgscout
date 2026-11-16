package application

import "github.com/firstpilotpirx/tgscout/components/matching/core/domain"

// SettingsRepository is the driven port that gives the core the settings of
// tgscout.yaml, read once at start.
type SettingsRepository interface {
	// Settings returns the resolved settings.
	Settings() domain.Settings
}
