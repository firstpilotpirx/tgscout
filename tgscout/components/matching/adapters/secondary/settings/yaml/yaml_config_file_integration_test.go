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

func TestCheckConfigFileMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bikes.yaml")

	err := yaml.CheckConfigFile(path)

	assert.ErrorIs(t, err, yaml.ErrConfigNotFound)
	assert.Contains(t, err.Error(), path)
}

func TestCheckConfigFileBrokenYamlNamesTheLine(t *testing.T) {
	path := writeConfig(t, "requests:\n  - id: bike\n    find: kids bike\n    chats: x: y\n")

	err := yaml.CheckConfigFile(path)

	var syntaxErr *yaml.ConfigSyntaxError
	require.True(t, errors.As(err, &syntaxErr), "got %v", err)
	assert.Equal(t, path, syntaxErr.Path)
	assert.Equal(t, 4, syntaxErr.Line)
	assert.Contains(t, syntaxErr.Reason, "mapping values are not allowed")
}

func TestCheckConfigFileValidYaml(t *testing.T) {
	path := writeConfig(t, "requests:\n  - id: bike\n    find: kids bike\n")

	assert.NoError(t, yaml.CheckConfigFile(path))
}
