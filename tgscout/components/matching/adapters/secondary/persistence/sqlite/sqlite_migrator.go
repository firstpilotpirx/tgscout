// Package sqlite keeps the messages, decisions and cursors of tgscout in SQLite.
package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrations holds the schema steps, one up and one down file each.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationResult tells what Migrate did.
type MigrationResult struct {
	Version uint // schema version after the run
	Tables  int  // tables of the app, without the migration bookkeeping
	Applied bool // false when the schema was already up to date
}

// Migrate applies the schema steps the database does not have yet. It leaves
// db open: the golang-migrate driver would close it on Close.
func Migrate(db *sql.DB) (MigrationResult, error) {
	source, err := iofs.New(Migrations, "migrations")
	if err != nil {
		return MigrationResult{}, fmt.Errorf("read migrations: %w", err)
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return MigrationResult{}, fmt.Errorf("prepare migrations: %w", err)
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("prepare migrations: %w", err)
	}
	applied := true
	if err := migrator.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return MigrationResult{}, fmt.Errorf("apply migrations: %w", err)
		}
		applied = false
	}
	return result(db, migrator, applied)
}

func result(db *sql.DB, migrator *migrate.Migrate, applied bool) (MigrationResult, error) {
	version, _, err := migrator.Version()
	if err != nil {
		return MigrationResult{}, fmt.Errorf("read schema version: %w", err)
	}
	var tables int
	err = db.QueryRow(`SELECT count(*) FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migrations'`).Scan(&tables)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("count tables: %w", err)
	}
	return MigrationResult{Version: version, Tables: tables, Applied: applied}, nil
}
