package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func TestBackoffDoublesUpToItsCap(t *testing.T) {
	backoff := domain.Backoff{Max: 4 * time.Second}

	assert.Equal(t, time.Second, backoff.Pause(1))
	assert.Equal(t, 2*time.Second, backoff.Pause(2))
	assert.Equal(t, 4*time.Second, backoff.Pause(3))
	assert.Equal(t, 4*time.Second, backoff.Pause(4))
}

func TestBackoffStaysAtItsCapForALargeTry(t *testing.T) {
	backoff := domain.Backoff{Max: time.Minute}

	assert.Equal(t, time.Minute, backoff.Pause(200))
	assert.Equal(t, time.Second, backoff.Pause(0))
}
