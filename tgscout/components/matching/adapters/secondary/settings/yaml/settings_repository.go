// Package yaml reads the settings of tgscout from tgscout.yaml once, at start.
package yaml

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// YamlSettingsRepository gives the core the resolved settings of one
// tgscout.yaml file.
type YamlSettingsRepository struct {
	settings domain.Settings
	warnings []string
}

// LoadSettings reads and checks the file at path. It returns an error
// wrapping ErrConfigNotFound, a *ConfigSyntaxError or a *ConfigErrors.
func LoadSettings(path string) (*YamlSettingsRepository, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	settings, warnings, err := Parse(data)
	var syntaxErr *ConfigSyntaxError
	var configErrs *ConfigErrors
	switch {
	case errors.As(err, &syntaxErr):
		syntaxErr.Path = path
		return nil, syntaxErr
	case errors.As(err, &configErrs):
		configErrs.Path = path
		return nil, configErrs
	case err != nil:
		return nil, err
	}
	return &YamlSettingsRepository{settings: settings, warnings: warnings}, nil
}

// Settings returns the resolved settings.
func (r *YamlSettingsRepository) Settings() domain.Settings { return r.settings }

// Warnings returns the problems that do not stop the start.
func (r *YamlSettingsRepository) Warnings() []string { return r.warnings }
