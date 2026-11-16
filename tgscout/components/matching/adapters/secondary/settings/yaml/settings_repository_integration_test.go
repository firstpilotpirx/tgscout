//go:build integration

package yaml_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/settings/yaml"
)

func writeConfig(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tgscout.yaml")
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
	return path
}

func TestLoadSettingsMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bikes.yaml")

	_, err := yaml.LoadSettings(path)

	assert.ErrorIs(t, err, yaml.ErrConfigNotFound)
	assert.Contains(t, err.Error(), path)
}

func TestLoadSettingsBrokenYamlNamesTheLine(t *testing.T) {
	path := writeConfig(t, "requests:\n  - id: bike\n    find: kids bike\n    chats: x: y\n")

	_, err := yaml.LoadSettings(path)

	var syntaxErr *yaml.ConfigSyntaxError
	require.True(t, errors.As(err, &syntaxErr), "got %v", err)
	assert.Equal(t, path, syntaxErr.Path)
	assert.Equal(t, 4, syntaxErr.Line)
	assert.Contains(t, syntaxErr.Reason, "mapping value is not allowed")
}

func TestLoadSettingsInvalidFieldsNameThePath(t *testing.T) {
	path := writeConfig(t, "requests: []\n")

	_, err := yaml.LoadSettings(path)

	var configErrs *yaml.ConfigErrors
	require.True(t, errors.As(err, &configErrs), "got %v", err)
	assert.Equal(t, path, configErrs.Path)
	assert.NotEmpty(t, configErrs.Errors)
}

func TestLoadSettingsValidFileGivesSettingsAndWarnings(t *testing.T) {
	path := writeConfig(t, manyChats(101))

	repository, err := yaml.LoadSettings(path)

	require.NoError(t, err)
	assert.Equal(t, "bike", repository.Settings().Requests[0].ID)
	assert.Equal(t, []string{"101 chats; the design maximum is 100"}, repository.Warnings())
}
