// Package sqlite opens the SQLite file of tgscout with mattn/go-sqlite3.
package sqlite

import (
	"database/sql"
	"fmt"
	"net/url"

	// Registers the "sqlite3" database/sql driver.
	_ "github.com/mattn/go-sqlite3"
)

// DSN is the data source name for the file at path: foreign keys on, WAL journal.
func DSN(path string) string {
	return "file:" + (&url.URL{Path: path}).EscapedPath() + "?_foreign_keys=on&_journal_mode=WAL"
}

// Open opens the SQLite file at path, creating it when it does not exist.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", DSN(path))
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close() // the ping error is the one to report
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	return db, nil
}
