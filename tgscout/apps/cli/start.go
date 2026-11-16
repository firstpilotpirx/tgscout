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
	if !checkConfig(opts.config, con) {
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
	_ = matching.NewMatchingConfig(matching.Ports{}) // the next slices pass their adapters here
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

func checkConfig(path string, con *console) bool {
	_, err := yaml.LoadSettings(path)
	var syntaxErr *yaml.ConfigSyntaxError
	switch {
	case err == nil:
		con.ok("config", path, "")
		return true
	case errors.Is(err, yaml.ErrConfigNotFound):
		con.fail("config file not found: "+path, "copy tgscout.yaml.example there, or pass --config <path>", path)
	case errors.As(err, &syntaxErr):
		con.fail("config is not valid YAML: "+path, lineHint(syntaxErr), path)
	default:
		con.fail(err.Error(), "")
	}
	return false
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
