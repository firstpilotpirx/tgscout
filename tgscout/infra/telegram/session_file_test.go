package telegram

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckModeAcceptsAPrivateFile(t *testing.T) {
	assert.NoError(t, checkMode("./tgscout.session", 0o600))
	assert.NoError(t, checkMode("./tgscout.session", 0o400))
}

func TestCheckModeRejectsAFileOpenToOthers(t *testing.T) {
	for _, mode := range []fs.FileMode{0o644, 0o640, 0o606} {
		err := checkMode("./tgscout.session", mode)

		var open *SessionOpenError
		require.True(t, errors.As(err, &open), "mode %o", mode)
		assert.Equal(t, "./tgscout.session", open.Path)
		assert.Equal(t, mode, open.Mode)
		assert.Contains(t, err.Error(), fmt.Sprintf("mode %o", mode))
	}
}
