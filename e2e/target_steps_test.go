package e2e_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cucumber/godog"
)

// The version 1 targets the target tests prove (design D9–D11).
const (
	latencyTargetE2E = 10 * time.Second
	catchUpScaled    = 15 * time.Second // 15 minutes, time scaled 60 times
	storeYearLimit   = 10 << 30         // 10 GB
	targetChats      = 100
	targetWorkers    = 4
	historyPerChat   = 500
	sizeMessages     = 100_000
	dayMessages      = 5_000
	yearRequests     = 10
)

var catchUpSummary = regexp.MustCompile(`catch-up done in (\S+) · [\s\S]*? (\d[\d,]*) judged`)

// fakePlay is one line of the fake Telegram's played file: an update it
// delivered live, and when (Unix nanoseconds).
type fakePlay struct {
	Chat string `json:"chat"`
	ID   int    `json:"id"`
	At   int64  `json:"at"`
}

// targetWorld is what one target scenario measured.
type targetWorld struct {
	started  time.Time
	messages int   // the live messages the load test plays
	perMsg   int64 // bytes the store grew per judged message
}

func targetSteps(sc *godog.ScenarioContext, w *world) {
	t := &targetWorld{}
	sc.Step(`^100 watched chats and a peak of 10 messages per second inside a day's flow, with a made-up LLM that answers in half a second$`,
		func() { t.messages = w.loadFlow() })
	sc.Step(`^the load test runs$`, func() error { return w.runLoad(t.messages) })
	sc.Step(`^p95 from message to send is at most 10 seconds$`, w.loadP95)
	sc.Step(`^100 chats with 500 missed messages each and a made-up LLM$`, w.missedHistory)
	sc.Step(`^tgscout starts$`, func() error {
		t.started = time.Now()
		return w.startMatching()
	})
	sc.Step(`^the catch-up summary comes within 15 minutes, scaled$`, func() error { return w.catchUpInTime(t.started) })
	sc.Step(`^a store filled with a year of decisions at 5,000 messages a day$`, func() error {
		grown, err := w.fillStore()
		t.perMsg = grown / sizeMessages
		return err
	})
	sc.Step(`^its size is measured$`, func() error {
		if t.perMsg <= 0 {
			return fmt.Errorf("the store grew %d bytes per message", t.perMsg)
		}
		return nil
	})
	sc.Step(`^it is under 10 GB$`, func() error {
		year := t.perMsg * yearRequests * dayMessages * 365
		fmt.Printf("size test: %d bytes per judged message, a year %.2f GB\n", t.perMsg, float64(year)/(1<<30))
		if year >= storeYearLimit {
			return fmt.Errorf("a year takes %d bytes (%d per message), want under 10 GB", year, t.perMsg)
		}
		return nil
	})
}

func targetChat(n int) string { return fmt.Sprintf("@made_up_shop_%03d", n) }

func (w *world) targetConfig(answer string, delay time.Duration) {
	w.request("kids-bike", bikeFind)
	w.lmStudio().answers[""] = answer
	w.lmStudio().setDelay(delay)
	w.matching.workers = targetWorkers
	w.matching.chats = nil
	for n := range targetChats {
		w.matching.chats = append(w.matching.chats, targetChat(n))
	}
}

// loadFlow plays 10 s of the day's flow scaled 20 times, 10 s at 10
// messages a second and 10 s of day flow again, over the 100 chats (design D9).
func (w *world) loadFlow() int {
	w.targetConfig("yes", 500*time.Millisecond)
	dayGap := int(86_400_000 / (dayMessages * 20)) // ms between two messages of the scaled day flow
	var updates []fakeUpdate
	add := func(gap, count int) {
		for range count {
			n := len(updates)
			updates = append(updates, fakeUpdate{Chat: targetChat(n % targetChats), ID: firstID + n,
				Text: fmt.Sprintf("Selling a 16-inch kids' bike, lot %d", n), AfterMs: gap})
		}
	}
	add(dayGap, 10_000/dayGap)
	add(100, 100)
	add(dayGap, 10_000/dayGap)
	w.tg.fake.Updates = updates
	return len(updates)
}

// runLoad starts tgscout and stops it once every message was sent.
func (w *world) runLoad(messages int) error {
	if err := w.startMatching(); err != nil {
		return err
	}
	if err := w.eventuallyWithin("a send of every message", time.Minute, func() bool {
		sends, err := w.sends()
		return err == nil && len(sends) == messages
	}); err != nil {
		return err
	}
	return w.run.signal(syscall.SIGINT)
}

// loadP95 is the p95 of send time minus play time over every message.
func (w *world) loadP95() error {
	plays, err := readLines[fakePlay](filepath.Join(w.dir, playedFileName))
	if err != nil {
		return err
	}
	playedAt := map[string]int64{}
	for _, each := range plays {
		playedAt[fmt.Sprintf("Selling a 16-inch kids' bike, lot %d", each.ID-firstID)] = each.At
	}
	sends, err := w.sends()
	if err != nil {
		return err
	}
	latencies := make([]time.Duration, 0, len(sends))
	for _, each := range sends {
		text := each.Text[strings.LastIndex(each.Text, "\n")+1:]
		at, ok := playedAt[text]
		if !ok {
			return fmt.Errorf("a send of a message never played: %q", text)
		}
		latencies = append(latencies, time.Duration(each.At-at))
	}
	slices.Sort(latencies)
	p95 := latencies[(len(latencies)*95+99)/100-1]
	fmt.Printf("load test: %d messages, p95 from message to send %s\n", len(latencies), p95)
	if p95 > latencyTargetE2E {
		return fmt.Errorf("p95 from message to send is %s, want at most %s", p95, latencyTargetE2E)
	}
	return nil
}

func (w *world) sends() ([]fakeSend, error) {
	return readLines[fakeSend](filepath.Join(w.dir, sentFileName))
}

// readLines reads a file of JSON lines; a missing file has none.
func readLines[T any](path string) ([]T, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: a file of the scenario's own folder
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	var lines []T
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		var value T
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
		}
		lines = append(lines, value)
	}
	return lines, nil
}

// missedHistory is 500 history messages in each of 100 chats, inside the
// 3-day default depth, and an LLM that answers no at once (design D10).
func (w *world) missedHistory() {
	w.targetConfig("no", 0)
	w.matching.extra += "history_pause: 17ms\n" // the default 1s, scaled 60 times (design D6)
	for n := range targetChats {
		for i := range historyPerChat {
			w.tg.fake.History = append(w.tg.fake.History, fakeHistory{Chat: targetChat(n), ID: firstID + i,
				Text: fmt.Sprintf("Selling a lamp, lot %d", i), AgeMinutes: 1 + i*5})
		}
	}
}

// catchUpInTime waits for the summary, then checks its time and count.
func (w *world) catchUpInTime(started time.Time) error {
	if err := w.eventuallyWithin("the catch-up summary", time.Minute, func() bool {
		return catchUpSummary.MatchString(w.run.stdout.String())
	}); err != nil {
		return err
	}
	came := time.Since(started)
	found := catchUpSummary.FindStringSubmatch(w.run.stdout.String())
	took, err := time.ParseDuration(found[1])
	if err != nil {
		return fmt.Errorf("the summary's time %q: %w", found[1], err)
	}
	judged, err := strconv.Atoi(strings.ReplaceAll(found[2], ",", ""))
	if err != nil {
		return fmt.Errorf("the summary's judged count %q: %w", found[2], err)
	}
	fmt.Printf("catch-up test: summary after %s, done in %s, %d judged\n", came.Round(time.Millisecond), took, judged)
	return errors.Join(
		atMost("the summary came", came, catchUpScaled), atMost("the summary's own time", took, catchUpScaled),
		equal("judged", judged, targetChats*historyPerChat), w.run.signal(syscall.SIGINT))
}

func atMost(what string, got, limit time.Duration) error {
	if got > limit {
		return fmt.Errorf("%s after %s, want at most %s", what, got, limit)
	}
	return nil
}

func equal(what string, got, want int) error {
	if got != want {
		return fmt.Errorf("%s %d, want %d", what, got, want)
	}
	return nil
}

// fillStore creates the store with one start and stop of tgscout, then puts
// 100,000 judged messages into it, each with one decision, the way the store
// writes them, and gives the growth of the file (design D11).
func (w *world) fillStore() (int64, error) {
	if err := w.writeConfig(validConfig); err != nil {
		return 0, err
	}
	if err := w.start(); err != nil {
		return 0, err
	}
	if err := errors.Join(w.run.signal(syscall.SIGINT), w.run.exited()); err != nil {
		return 0, err
	}
	before, err := w.storeSize()
	if err != nil {
		return 0, err
	}
	db, err := sql.Open("sqlite3", w.dbPath())
	if err != nil {
		return 0, fmt.Errorf("open the store: %w", err)
	}
	defer func() { _ = db.Close() }() // a read-only measure; the size is already taken
	if err := insertJudged(db); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return 0, fmt.Errorf("checkpoint the store: %w", err)
	}
	after, err := w.storeSize()
	return after - before, err
}

func insertJudged(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("fill the store: %w", err)
	}
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for n := range sizeMessages {
		chat, message, when := int64(-1_000_000_000_000-n%targetChats), firstID+n, at.Add(time.Duration(n)*time.Second)
		_, err = tx.Exec(`INSERT INTO processed_message (chat_id, message_id, processed_at) VALUES (?, ?, ?)`,
			chat, message, when)
		if err == nil {
			_, err = tx.Exec(`INSERT INTO decision (chat_id, message_id, request_id, matched, decided_at)
				VALUES (?, ?, ?, ?, ?)`, chat, message, "kids-bike", false, when)
		}
		if err != nil {
			return errors.Join(fmt.Errorf("fill the store: %w", err), tx.Rollback())
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fill the store: %w", err)
	}
	return nil
}

// storeSize is the size of the store with its WAL file.
func (w *world) storeSize() (int64, error) {
	var size int64
	for _, path := range []string{w.dbPath(), w.dbPath() + "-wal"} {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("store size: %w", err)
		}
		size += info.Size()
	}
	return size, nil
}
