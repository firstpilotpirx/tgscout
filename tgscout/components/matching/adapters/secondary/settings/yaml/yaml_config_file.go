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
		return viperSyntaxError(path, err)
	}
	return nil
}

func viperSyntaxError(path string, err error) *ConfigSyntaxError {
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
