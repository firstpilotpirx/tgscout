// Package yaml reads the settings of tgscout from tgscout.yaml. This slice
// checks only that the file exists and is valid YAML; the hierarchical-settings
// slice adds YamlSettingsRepository and the fields.
package yaml

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/viper"
)

// ErrConfigNotFound means there is no config file at the given path.
var ErrConfigNotFound = errors.New("config file not found")

// ConfigSyntaxError is a config file that is not valid YAML.
type ConfigSyntaxError struct {
	Path   string
	Line   int
	Reason string
}

func (e *ConfigSyntaxError) Error() string {
	if e.Line == 0 {
		return fmt.Sprintf("config is not valid YAML: %s: %s", e.Path, e.Reason)
	}
	return fmt.Sprintf("config is not valid YAML: %s: line %d: %s", e.Path, e.Line, e.Reason)
}

// yamlLine finds "line N: reason" in the error text of the YAML parser.
var yamlLine = regexp.MustCompile(`line (\d+): (.+)$`)

// CheckConfigFile checks that the file at path exists and is valid YAML.
func CheckConfigFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	reader := viper.New()
	reader.SetConfigFile(path)
	reader.SetConfigType("yaml")
	if err := reader.ReadInConfig(); err != nil {
		return syntaxError(path, err)
	}
	return nil
}

func syntaxError(path string, err error) *ConfigSyntaxError {
	match := yamlLine.FindStringSubmatch(err.Error())
	if match == nil {
		return &ConfigSyntaxError{Path: path, Reason: err.Error()}
	}
	line, convErr := strconv.Atoi(match[1])
	if convErr != nil {
		return &ConfigSyntaxError{Path: path, Reason: err.Error()}
	}
	return &ConfigSyntaxError{Path: path, Line: line, Reason: match[2]}
}
