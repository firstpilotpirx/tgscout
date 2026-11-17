package telegram

import (
	"context"
	"sync"

	"github.com/gotd/td/session"
)

// FileStorage is gotd's session storage over the session file. On a first
// run it keeps what gotd stores in memory until Commit, so no file exists
// before the sign-in succeeds; gotd's own file storage writes it with mode 600.
type FileStorage struct {
	file      session.FileStorage
	mu        sync.Mutex
	held      bool
	pending   []byte
	committed bool
}

// NewFileStorage opens the storage; firstRun holds every write until Commit.
func NewFileStorage(path string, firstRun bool) *FileStorage {
	return &FileStorage{file: session.FileStorage{Path: path}, held: firstRun}
}

// LoadSession returns the held session on a first run, else the file.
func (s *FileStorage) LoadSession(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held && !s.committed {
		if s.pending == nil {
			return nil, session.ErrNotFound
		}
		return append([]byte(nil), s.pending...), nil
	}
	return s.file.LoadSession(ctx)
}

// StoreSession holds the data on a first run before Commit, else writes it.
func (s *FileStorage) StoreSession(ctx context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held && !s.committed {
		s.pending = append([]byte(nil), data...)
		return nil
	}
	return s.file.StoreSession(ctx, data)
}

// Commit writes the held session and lets every later write through.
func (s *FileStorage) Commit(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.committed = true
	if s.pending == nil {
		return nil
	}
	data := s.pending
	s.pending = nil
	return s.file.StoreSession(ctx, data)
}
