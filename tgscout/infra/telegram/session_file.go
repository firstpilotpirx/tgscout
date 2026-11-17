package telegram

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// SessionOpenError means users other than the owner may read or write the
// session file.
type SessionOpenError struct {
	Path string
	Mode fs.FileMode
}

func (e *SessionOpenError) Error() string {
	return fmt.Sprintf("the session file %s is open to other users (mode %o)", e.Path, e.Mode)
}

// CheckSessionFile tells whether the session file exists; an existing file
// must be private to its owner.
func CheckSessionFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("session file %s: %w", path, err)
	}
	return true, checkMode(path, info.Mode().Perm())
}

// checkMode refuses any group or other permission bit.
func checkMode(path string, mode fs.FileMode) error {
	if mode&0o077 != 0 {
		return &SessionOpenError{Path: path, Mode: mode}
	}
	return nil
}
