package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/firstpilotpirx/tgscout/components/matching"
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

// start walks the start flow: config, database, wait for a stop, close.
func start(ctx context.Context, opts options, con *console) int {
	settings, ok := loadConfig(opts.config, con)
	if !ok {
		return exitError
	}
	path := databasePath(opts.config, opts.db)
	_, statErr := os.Stat(path)
	existed := statErr == nil
	db, err := sqlitedb.Open(path)
	if err != nil {
		con.fail(fmt.Sprintf("database %s: %v", path, err), "", path)
		return exitError
	}
	result, err := sqlite.Migrate(db)
	if err != nil {
		_ = db.Close() // the migration error is the one to report
		con.fail(fmt.Sprintf("database %s: %v", path, err), "", path)
		return exitError
	}
	con.ok("database", path, databaseDetail(result, existed))
	_ = matching.NewMatchingConfig(matching.Ports{Settings: settings}) // the next slices pass their other adapters here
	con.running()
	<-ctx.Done()
	con.step("stopping…")
	if err := db.Close(); err != nil {
		con.fail(fmt.Sprintf("database %s: close: %v", path, err), "", path)
		return exitError
	}
	con.done("stopped")
	return exitOK
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
