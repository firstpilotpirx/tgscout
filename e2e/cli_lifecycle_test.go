package e2e_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/cucumber/godog"
	_ "github.com/mattn/go-sqlite3"
)

// The world of one scenario: a temp folder with the config, the database and
// the last tgscout run.
type world struct {
	dir      string
	config   string
	run      *process
	schema   string
	tg       telegramWorld
	matching matchingWorld
	echo     bool // a run on a terminal shows what the owner types
}

// validConfig is a made-up config with every required field.
const validConfig = "owner: '@made_up_owner'\nlm_studio:\n  model: example-model\n" +
	"requests:\n  - id: bike\n    find: a kids bike, 16 inch, under 100 EUR\n    chats: ['@made_up_market']\n"

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format: "pretty", Paths: []string{"features"}, Tags: "~@wip",
			Strict: true, TestingT: t, Concurrency: 4,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("scenarios failed")
	}
}

func initializeScenario(sc *godog.ScenarioContext) {
	w := &world{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		dir, err := os.MkdirTemp("", "tgscout-e2e-")
		w.dir, w.config, w.tg, w.echo = dir, "tgscout.yaml", defaultTelegram(), false
		w.matching = matchingWorld{}
		return ctx, err
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
		if w.run != nil {
			w.run.stop()
		}
		if w.matching.lm != nil {
			w.matching.lm.server.Close()
		}
		return ctx, errors.Join(err, os.RemoveAll(w.dir))
	})
	givenSteps(sc, w)
	whenSteps(sc, w)
	thenSteps(sc, w)
	settingsSteps(sc, w)
	telegramSteps(sc, w)
	matchingSteps(sc, w)
}

func givenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^no config file at the path the owner gives$`, func() { w.config = "bikes.yaml" })
	sc.Step(`^no config file at all$`, func() { w.config = "" })
	sc.Step(`^a config file with broken YAML on its fourth line$`, func() error {
		return w.writeConfig("requests:\n  - id: bike\n    find: a kids bike\n    chats: a: b\n")
	})
	sc.Step(`^a valid config file( and no database file yet)?$`, func() error { return w.writeConfig(validConfig) })
	sc.Step(`^a database created by an earlier start, with rows in it$`, w.earlierStartWithRows)
	sc.Step(`^tgscout is running with a valid config file$`, func() error {
		if err := w.writeConfig(validConfig); err != nil {
			return err
		}
		return w.start()
	})
}

func whenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the owner starts tgscout( again)?$`, func() error { return w.start() })
	sc.Step(`^the owner starts tgscout and then presses Ctrl\+C$`, func() error {
		if err := w.start(); err != nil {
			return err
		}
		return w.run.signal(syscall.SIGINT)
	})
	sc.Step(`^the system asks tgscout to terminate$`, func() error { return w.run.signal(syscall.SIGTERM) })
	sc.Step(`^the owner asks tgscout for help$`, func() error { return w.start("--help") })
	sc.Step(`^the owner starts tgscout with an option it does not know$`, func() error { return w.start("--colour") })
}

func (w *world) writeConfig(text string) error {
	return os.WriteFile(filepath.Join(w.dir, w.config), []byte(text), 0o600)
}

// start runs tgscout in the scenario folder with the scenario's config.
func (w *world) start(extra ...string) error {
	args := extra
	if w.config != "" {
		args = append([]string{"--config", "./" + w.config}, extra...)
	}
	env, err := w.tg.write(w.dir)
	if err != nil {
		return err
	}
	run, err := startProcess(w.dir, env, args...)
	if err != nil {
		return err
	}
	w.run = run
	return run.settle()
}

func (w *world) earlierStartWithRows() error {
	if err := w.writeConfig(validConfig); err != nil {
		return err
	}
	if err := w.start(); err != nil {
		return err
	}
	if err := w.run.signal(syscall.SIGINT); err != nil {
		return err
	}
	db, err := w.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`INSERT INTO decision (chat_id, message_id, request_id, matched, decided_at)
		VALUES (1001, 7, 'bike', 1, CURRENT_TIMESTAMP)`); err != nil {
		return fmt.Errorf("insert a row: %w", err)
	}
	w.schema, err = schemaOf(db)
	return err
}

func (w *world) dbPath() string { return filepath.Join(w.dir, "tgscout.db") }

func (w *world) openDB() (*sql.DB, error) {
	if _, err := os.Stat(w.dbPath()); err != nil {
		return nil, fmt.Errorf("database file: %w", err)
	}
	return sql.Open("sqlite3", "file:"+w.dbPath())
}

func schemaOf(db *sql.DB) (string, error) {
	rows, err := db.Query(`SELECT name || ': ' || coalesce(sql, '') FROM sqlite_master ORDER BY name`)
	if err != nil {
		return "", fmt.Errorf("read schema: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", fmt.Errorf("read schema: %w", err)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), rows.Err()
}
