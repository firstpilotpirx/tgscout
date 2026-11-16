package yaml_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/settings/yaml"
)

func TestConfigSyntaxErrorNamesPathAndLine(t *testing.T) {
	err := &yaml.ConfigSyntaxError{Path: "./tgscout.yaml", Line: 4, Reason: "bad"}

	assert.Equal(t, "config is not valid YAML: ./tgscout.yaml: line 4: bad", err.Error())
}

func TestConfigSyntaxErrorWithoutLine(t *testing.T) {
	err := &yaml.ConfigSyntaxError{Path: "./tgscout.yaml", Reason: "bad"}

	assert.Equal(t, "config is not valid YAML: ./tgscout.yaml: bad", err.Error())
}
