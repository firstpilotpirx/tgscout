package e2e_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/cucumber/godog"
)

func thenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^tgscout prints an error that names the missing path$`, func() error {
		return w.stderrHas("config file not found: ./bikes.yaml")
	})
	sc.Step(`^tgscout prints an error that names line 4$`, func() error { return w.stderrHas("line 4:") })
	sc.Step(`^it exits with a non-zero code$`, func() error { return w.exitCode(func(c int) bool { return c != 0 }) })
	sc.Step(`^it exits with code (\d+)$`, func(want int) error { return w.exitCode(func(c int) bool { return c == want }) })
	sc.Step(`^no database file is created$`, func() error {
		if _, err := os.Stat(w.dbPath()); !os.IsNotExist(err) {
			return fmt.Errorf("a database file exists: %v", err)
		}
		return nil
	})
	sc.Step(`^the database file exists next to the config file$`, func() error {
		_, err := os.Stat(w.dbPath())
		return err
	})
	sc.Step(`^(?:it|the database) holds the chat cursor, processed message and decision tables$`, w.holdsTables)
	sc.Step(`^the console says the database was created$`, func() error { return w.stdoutHas("tgscout.db created") })
	sc.Step(`^the rows are still there$`, w.rowsStillThere)
	sc.Step(`^the schema is unchanged$`, w.schemaUnchanged)
	sc.Step(`^the console says the schema is up to date$`, func() error { return w.stdoutHas("schema up to date") })
	sc.Step(`^tgscout says that it stopped$`, func() error { return w.stdoutHas("[ok] stopped") })
	sc.Step(`^tgscout prints the usage with every option and its default$`, func() error {
		return w.stdoutHas("tgscout [flags]", `--config string`, `(default "./tgscout.yaml")`, "--db string",
			"tgscout.db next to the config", "-h, --help")
	})
	sc.Step(`^tgscout prints the error and the usage$`, func() error {
		return w.stderrHas("unknown flag: --colour", "tgscout [flags]")
	})
}

func has(stream, text string, want []string) error {
	for _, each := range want {
		if !strings.Contains(text, each) {
			return fmt.Errorf("%s has no %q; it holds:\n%s", stream, each, text)
		}
	}
	return nil
}

func (w *world) stdoutHas(want ...string) error { return has("stdout", w.run.stdout.String(), want) }
func (w *world) stderrHas(want ...string) error { return has("stderr", w.run.stderr.String(), want) }

func (w *world) exitCode(ok func(int) bool) error {
	if err := w.run.exited(); err != nil {
		return err
	}
	if !ok(w.run.code) {
		return fmt.Errorf("exit code %d; output:\n%s", w.run.code, w.run.output())
	}
	return nil
}

func (w *world) holdsTables() error {
	db, err := w.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	for _, table := range []string{"chat_cursor", "processed_message", "decision"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			return fmt.Errorf("table %s: %w", table, err)
		}
	}
	return nil
}

func (w *world) rowsStillThere() error {
	db, err := w.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM decision WHERE chat_id = 1001 AND message_id = 7`).Scan(&count); err != nil {
		return fmt.Errorf("read rows: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("want the earlier row, found %d", count)
	}
	return nil
}

func (w *world) schemaUnchanged() error {
	db, err := w.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	now, err := schemaOf(db)
	if err != nil {
		return err
	}
	if now != w.schema {
		return fmt.Errorf("schema changed:\nbefore:\n%s\nnow:\n%s", w.schema, now)
	}
	return nil
}
