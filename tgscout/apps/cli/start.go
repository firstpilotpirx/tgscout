package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/settings/yaml"
	sqlitedb "github.com/firstpilotpirx/tgscout/infra/sqlite"
)

// databasePath is the --db value, or tgscout.db in the folder of the config file.
func databasePath(config, db string) string {
	if db != "" {
		return db
	}
	return filepath.Dir(config) + string(filepath.Separator) + "tgscout.db"
}

// start walks the start flow: config, database, Telegram, wait for a stop, close.
func start(ctx context.Context, opts options, con *console, deps startDeps) int {
	repository, ok := loadConfig(opts.config, con)
	if !ok {
		return exitError
	}
	path := databasePath(opts.config, opts.db)
	db, ok := openDatabase(path, con)
	if !ok {
		return exitError
	}
	matcher := newLive(repository, db, deps.sleep)
	known := knownPeers{repository: sqlite.NewSqlitePeerRepository(db), path: path}
	if watch(ctx, con, deps, opts.config, matcher, known) != exitOK {
		_ = db.Close() // the Telegram error is the one to report
		return exitError
	}
	if matcher.announcedStop() {
		con.statusTotal(matcher.total())
	} else {
		con.step("stopping…") // stopped before the workers ran
	}
	if err := db.Close(); err != nil {
		con.fail(fmt.Sprintf("database %s: close: %v", path, err), "", path)
		return exitError
	}
	con.stopped(matcher.queued())
	return exitOK
}

// openDatabase opens and migrates the SQLite file, printing what it did.
func openDatabase(path string, con *console) (*sql.DB, bool) {
	_, statErr := os.Stat(path)
	existed := statErr == nil
	db, err := sqlitedb.Open(path)
	if err != nil {
		con.fail(fmt.Sprintf("database %s: %v", path, err), "", path)
		return nil, false
	}
	result, err := sqlite.Migrate(db)
	if err != nil {
		_ = db.Close() // the migration error is the one to report
		con.fail(fmt.Sprintf("database %s: %v", path, err), "", path)
		return nil, false
	}
	con.ok("database", path, databaseDetail(result, existed))
	return db, true
}

// loadConfig reads tgscout.yaml and prints its warnings and summary, or
// every problem it has.
func loadConfig(path string, con *console) (*yaml.YamlSettingsRepository, bool) {
	repository, err := yaml.LoadSettings(path)
	var syntaxErr *yaml.ConfigSyntaxError
	var configErrs *yaml.ConfigErrors
	switch {
	case err == nil:
		for _, warning := range repository.Warnings() {
			con.warn(path, warning)
		}
		settings := repository.Settings()
		con.summary(path, len(settings.Requests), settings.DistinctChats(), settings.LmStudio.Model)
		if !settings.LmStudio.Local() {
			con.nonLocalLlm(settings.LmStudio.URL)
		}
		return repository, true
	case errors.Is(err, yaml.ErrConfigNotFound):
		con.fail("config file not found: "+path, "copy tgscout.yaml.example there, or pass --config <path>", path)
	case errors.As(err, &syntaxErr):
		con.fail("config is not valid YAML: "+path, lineHint(syntaxErr), path)
	case errors.As(err, &configErrs):
		con.failList(path, listRows(configErrs.Errors))
	default:
		con.fail(err.Error(), "")
	}
	return nil, false
}

func listRows(errs []yaml.FieldError) []listRow {
	rows := make([]listRow, 0, len(errs))
	for _, each := range errs {
		rows = append(rows, listRow{line: each.Line, field: each.Field, reason: each.Reason})
	}
	return rows
}

func lineHint(err *yaml.ConfigSyntaxError) string {
	if err.Line == 0 {
		return err.Reason
	}
	return fmt.Sprintf("line %d: %s", err.Line, err.Reason)
}

func databaseDetail(result sqlite.MigrationResult, existed bool) string {
	if !existed {
		return fmt.Sprintf("created · migration %d · %d tables", result.Version, result.Tables)
	}
	if result.Applied {
		return fmt.Sprintf("migrated · migration %d", result.Version)
	}
	return fmt.Sprintf("schema up to date · migration %d", result.Version)
}
