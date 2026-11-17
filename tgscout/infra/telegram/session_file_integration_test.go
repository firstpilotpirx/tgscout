//go:build integration

package telegram

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSessionFileSaysWhenThereIsNone(t *testing.T) {
	exists, err := CheckSessionFile(filepath.Join(t.TempDir(), "tgscout.session"))

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckSessionFileAcceptsAPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tgscout.session")
	require.NoError(t, os.WriteFile(path, []byte("made-up"), 0o600))

	exists, err := CheckSessionFile(path)

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckSessionFileRejectsAnOpenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tgscout.session")
	require.NoError(t, os.WriteFile(path, []byte("made-up"), 0o600))
	require.NoError(t, os.Chmod(path, 0o644))

	_, err := CheckSessionFile(path)

	var open *SessionOpenError
	require.True(t, errors.As(err, &open))
	assert.Equal(t, os.FileMode(0o644), open.Mode)
}
