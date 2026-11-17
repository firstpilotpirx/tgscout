package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func TestP95HasNoValueWithoutSamples(t *testing.T) {
	_, ok := domain.P95(nil)

	assert.False(t, ok)
}

func TestP95OfOneSampleIsThatSample(t *testing.T) {
	got, ok := domain.P95([]time.Duration{3 * time.Second})

	assert.True(t, ok)
	assert.Equal(t, 3*time.Second, got)
}

func TestP95TakesTheNearestRank(t *testing.T) {
	var samples []time.Duration
	for i := 20; i >= 1; i-- { // unsorted on purpose
		samples = append(samples, time.Duration(i)*time.Second)
	}

	got, _ := domain.P95(samples)

	assert.Equal(t, 19*time.Second, got) // rank ceil(0.95*20) = 19
	assert.Equal(t, 20*time.Second, samples[0], "P95 leaves the caller's slice as it was")
}

func TestRunStatsCutGivesTheLastIntervalAndStartsANewOne(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	stats := domain.NewRunStats(start)
	stats.In()
	stats.In()
	latency := 2 * time.Second
	stats.Judged(&latency)
	stats.Judged(nil)
	stats.Matched()
	stats.LlmCall(500 * time.Millisecond)

	first := stats.Cut(7, start.Add(5*time.Minute))

	assert.Equal(t, domain.Status{In: 2, Judged: 2, Matches: 1, Queue: 7, Took: 5 * time.Minute,
		Latency: 2 * time.Second, HasLatency: true, LlmCall: 500 * time.Millisecond, HasLlmCall: true}, first)
	second := stats.Cut(0, start.Add(10*time.Minute))
	assert.Equal(t, domain.Status{Took: 5 * time.Minute}, second)
}

func TestRunStatsTotalCoversTheWholeRun(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	stats := domain.NewRunStats(start)
	for i := 1; i <= 3; i++ {
		latency := time.Duration(i) * time.Second
		stats.In()
		stats.Judged(&latency)
		stats.Cut(0, start.Add(time.Duration(i)*time.Minute))
	}

	total := stats.Total(1, start.Add(time.Hour))

	assert.Equal(t, 3, total.In)
	assert.Equal(t, 3, total.Judged)
	assert.Equal(t, time.Hour, total.Took)
	assert.Equal(t, 3*time.Second, total.Latency)
	assert.False(t, total.HasLlmCall)
	assert.Equal(t, 1, total.Queue)
}

func TestNilRunStatsDoesNothing(t *testing.T) {
	var stats *domain.RunStats
	latency := time.Second

	assert.NotPanics(t, func() {
		stats.In()
		stats.Judged(&latency)
		stats.Matched()
		stats.LlmCall(time.Second)
	})
}

func TestLmStudioLocalKnowsTheHostsOfThisMachine(t *testing.T) {
	for url, local := range map[string]bool{
		"http://localhost:1234/v1":    true,
		"http://LOCALHOST:1234/v1":    true,
		"http://127.0.0.1:1234/v1":    true,
		"http://[::1]:1234/v1":        true,
		"https://llm.example.net/v1":  false,
		"http://192.168.1.20:1234/v1": false,
		"http://localhost.example/v1": false,
	} {
		assert.Equal(t, local, domain.LmStudio{URL: url}.Local(), url)
	}
}
