package matching_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching"
)

func TestNewMatchingConfigWiresBothUseCases(t *testing.T) {
	config := matching.NewMatchingConfig(matching.Ports{})

	assert.NotNil(t, config.CollectMessage)
	assert.NotNil(t, config.MatchMessage)
}
