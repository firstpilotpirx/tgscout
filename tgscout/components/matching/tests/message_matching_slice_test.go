package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cucumber/godog"

	collect "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/collect/telegram"
	match "github.com/firstpilotpirx/tgscout/components/matching/adapters/primary/match/channel"
	queue "github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/queue/channel"
	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

func TestMessageMatchingSlice(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			s := newSlice()
			s.steps(sc)
			s.ownerSteps(sc)
		},
		Options: &godog.Options{
			Format: "pretty", Paths: []string{"features/slice"}, Tags: "~@wip", Strict: true, TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("slice scenarios failed")
	}
}

const watchedChatID = 42

// slice wires the real use cases with made-up secondary adapters.
type slice struct {
	repository *memoryRepository
	llm        *slowLlm
	queue      *queue.ChannelMessageQueue
	puts       countingQueue
	workers    int
	settings   domain.Settings
}

func newSlice() *slice {
	ref, _ := domain.NewChatRef("@made_up_market")
	return &slice{
		repository: newMemoryRepository(),
		llm:        &slowLlm{},
		queue:      queue.NewChannelMessageQueue(1000),
		workers:    2,
		settings: domain.Settings{Requests: []domain.Request{
			{ID: "bike", Find: "a kids bike", Chats: []domain.RequestChat{{Ref: ref}}},
		}},
	}
}

func (s *slice) steps(sc *godog.ScenarioContext) {
	sc.Step(`^a watched chat$`, func() {})
	sc.Step(`^a queue size of 1, one worker and an LLM that answers slowly$`, func() {
		s.queue, s.workers, s.llm.delay = queue.NewChannelMessageQueue(1), 1, 30*time.Millisecond
	})
	sc.Step(`^someone joins the chat$`, func() error {
		// a text too, so only the service flag drops it
		return s.play(telegram.Update{ChatID: watchedChatID, MessageID: 1, Text: "Made Up Reader joined", Service: true})
	})
	sc.Step(`^a photo without a caption appears in it$`, func() error {
		return s.play(telegram.Update{ChatID: watchedChatID, MessageID: 1})
	})
	sc.Step(`^a photo with a caption appears in it$`, func() error {
		return s.play(telegram.Update{ChatID: watchedChatID, MessageID: 1, Text: "Kids bike, 16 inch, see the photo"})
	})
	sc.Step(`^three new messages appear in a watched chat$`, func() error {
		return s.play(
			telegram.Update{ChatID: watchedChatID, MessageID: 1, Text: "first"},
			telegram.Update{ChatID: watchedChatID, MessageID: 2, Text: "second"},
			telegram.Update{ChatID: watchedChatID, MessageID: 3, Text: "third"},
		)
	})
	sc.Step(`^the LLM is not asked and nothing is saved$`, func() error {
		if err := s.notAsked(); err != nil {
			return err
		}
		return s.repository.empty()
	})
	sc.Step(`^the LLM is not asked$`, s.notAsked)
	sc.Step(`^the LLM is asked about the caption$`, func() error {
		return s.askedAbout("Kids bike, 16 inch, see the photo")
	})
	sc.Step(`^all three are judged, in the order they appeared$`, func() error {
		return s.askedAbout("first", "second", "third")
	})
}

// play hands the updates to the collector while the workers run, then waits
// until the queue is empty and the last judgment is done.
func (s *slice) play(updates ...telegram.Update) error {
	tracker := domain.NewMessageTracker()
	s.puts.queue = s.queue
	collectUseCase := application.NewCollectMessage(staticSettings{s.settings}, s.repository, &s.puts, tracker)
	matchUseCase := application.NewMatchMessage(s.repository, s.llm, nopNotifier{}, tracker)
	collector := collect.NewTelegramCollector(map[int64]collect.Chat{watchedChatID: {Ref: s.settings.Requests[0].Chats[0].Ref,
		Username: "made_up_market"}}, collectUseCase, func(domain.Message, error) {})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		match.RunWorkers(ctx, s.workers, s.queue, matchUseCase, nopReporter{})
		close(stopped)
	}()
	for _, update := range updates {
		collector.Handle(ctx, update)
	}
	deadline := time.Now().Add(2 * time.Second)
	for s.repository.processedCount() < s.textCount(updates) {
		if time.Now().After(deadline) {
			return fmt.Errorf("only %d of the messages were judged", s.repository.processedCount())
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-stopped
	return nil
}

func (s *slice) textCount(updates []telegram.Update) int {
	count := 0
	for _, update := range updates {
		if !update.Service && update.Text != "" {
			count++
		}
	}
	return count
}

func (s *slice) notAsked() error {
	if puts := s.puts.count(); puts > 0 {
		return fmt.Errorf("%d messages were queued for the LLM", puts)
	}
	if calls := s.llm.texts(); len(calls) > 0 {
		return fmt.Errorf("the LLM was asked about %q", calls)
	}
	return nil
}

func (s *slice) askedAbout(texts ...string) error {
	calls := s.llm.texts()
	if fmt.Sprint(calls) != fmt.Sprint(texts) {
		return fmt.Errorf("the LLM was asked about %q, want %q", calls, texts)
	}
	return nil
}

type staticSettings struct{ settings domain.Settings }

func (s staticSettings) Settings() domain.Settings { return s.settings }

type slowLlm struct {
	mu    sync.Mutex
	delay time.Duration
	calls []string
}

func (l *slowLlm) Judge(ctx context.Context, _, text string) (domain.Judgment, error) {
	l.mu.Lock()
	l.calls = append(l.calls, text)
	delay := l.delay
	l.mu.Unlock()
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return domain.Judgment{}, ctx.Err()
	}
	return domain.Judgment{Raw: "no"}, nil
}

func (l *slowLlm) texts() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.calls...)
}

type nopNotifier struct{}

func (nopNotifier) Notify(context.Context, domain.Match) error { return nil }

type nopReporter struct{}

func (nopReporter) Unclear(domain.QueueItem, application.Unclear) {}
func (nopReporter) Failed(domain.QueueItem, error)                {}

type memoryRepository struct {
	mu        sync.Mutex
	processed map[domain.MessageKey]bool
	sent      map[domain.MessageKey]int
	cursor    map[int64]int
	decisions int
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{processed: map[domain.MessageKey]bool{}, sent: map[domain.MessageKey]int{},
		cursor: map[int64]int{}}
}

func (r *memoryRepository) IsProcessed(_ context.Context, key domain.MessageKey) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.processed[key], nil
}

func (r *memoryRepository) SentRequests(context.Context, domain.MessageKey) ([]string, error) {
	return nil, nil
}

func (r *memoryRepository) SaveDecision(context.Context, domain.Decision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decisions++
	return nil
}

func (r *memoryRepository) MarkSent(_ context.Context, key domain.MessageKey, _ string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent[key]++
	return nil
}

func (r *memoryRepository) MarkProcessed(_ context.Context, key domain.MessageKey, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processed[key] = true
	return nil
}

func (r *memoryRepository) MoveCursor(_ context.Context, chatID int64, messageID int, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cursor[chatID] = max(r.cursor[chatID], messageID)
	return nil
}

// untouched tells whether the message has no send, no processed mark and
// the cursor of its chat has not reached it.
func (r *memoryRepository) untouched(key domain.MessageKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sent[key] > 0 || r.processed[key] || r.cursor[key.ChatID] >= key.MessageID {
		return fmt.Errorf("sent %d, processed %t, cursor %d", r.sent[key], r.processed[key], r.cursor[key.ChatID])
	}
	return nil
}

func (r *memoryRepository) processedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.processed)
}

func (r *memoryRepository) empty() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.decisions > 0 || len(r.processed) > 0 {
		return fmt.Errorf("%d decisions and %d processed marks were saved", r.decisions, len(r.processed))
	}
	return nil
}

// countingQueue counts what the collect use case puts in the real queue, so a
// message wrongly queued shows even when no worker took it yet.
type countingQueue struct {
	queue *queue.ChannelMessageQueue
	mu    sync.Mutex
	puts  int
}

func (q *countingQueue) Put(ctx context.Context, item domain.QueueItem) error {
	q.mu.Lock()
	q.puts++
	q.mu.Unlock()
	return q.queue.Put(ctx, item)
}

func (q *countingQueue) count() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.puts
}
