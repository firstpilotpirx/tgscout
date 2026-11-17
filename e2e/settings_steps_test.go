package e2e_test

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cucumber/godog"
)

// header holds the required root fields of a made-up config.
const header = "owner: '@made_up_owner'\nlm_studio:\n  model: example-model\n"

const bikeRequest = "  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market']\n"

// configs are the made-up configs the settings scenarios name.
var configs = map[string]string{
	"no requests":             header,
	"a request without chats": header + "requests:\n  - id: bike\n    find: a kids bike\n    chats: []\n",
	"two requests with one id": header + "requests:\n" + bikeRequest +
		"  - id: bike\n    find: a desk lamp\n    chats: ['@made_up_street']\n",
	"an unknown key":    header + "colour: red\nrequests:\n" + bikeRequest,
	"a zero depth":      header + "catch_up: {days: 0}\nrequests:\n" + bikeRequest,
	"an api_hash key":   header + "api_hash: made-up\nrequests:\n" + bikeRequest,
	"a zero queue size": header + "queue_size: 0\nrequests:\n" + bikeRequest,
	"a request without chats, a second request with the same id and an api_hash key": header +
		"requests:\n  - id: bike\n    find: a kids bike\n    chats: []\n" + bikeRequest + "api_hash: made-up\n",
	"a request whose chats are a username, a public chat link and a numeric ID": header +
		"requests:\n  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market', 'https://t.me/made_up_street', -1001234567890]\n",
	"101 distinct chats": header + "requests:\n  - id: bike\n    find: a kids bike\n    chats: [" + numericChats(101) + "]\n",
	"2 requests over 3 distinct chats and a model named example-model": header + "requests:\n" + bikeRequest +
		"  - id: lamp\n    find: a desk lamp\n    chats: ['@made_up_market', '@made_up_street', '-1001']\n",
}

func numericChats(count int) string {
	chats := make([]string, 0, count)
	for i := range count {
		chats = append(chats, fmt.Sprintf("'%d'", 1000+i))
	}
	return strings.Join(chats, ", ")
}

// errorLine is one line of the error list: "line <n>  <field>  <reason>".
var errorLine = regexp.MustCompile(`(?m)^\s+line\s+\d+\s+\S+\s+\S`)

func settingsSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^a config with (.+)$`, func(problem string) error {
		text, ok := configs[problem]
		if !ok {
			return fmt.Errorf("no made-up config for %q", problem)
		}
		return w.writeConfig(text)
	})
	sc.Step(`^tgscout prints an error naming (\S+) and its line$`, func(field string) error {
		pattern := regexp.MustCompile(`(?m)^\s+line\s+\d+\s+` + regexp.QuoteMeta(field) + `\s`)
		if !pattern.MatchString(w.run.stderr.String()) {
			return fmt.Errorf("stderr names no line of %s; it holds:\n%s", field, w.run.stderr.String())
		}
		return nil
	})
	sc.Step(`^tgscout says the config has (\d+) errors$`, func(n int) error {
		return w.stderrHas(fmt.Sprintf("tgscout.yaml: %d errors", n))
	})
	sc.Step(`^it prints one line for each of the three problems, each with its line$`, func() error {
		lines := errorLine.FindAllString(w.run.stderr.String(), -1)
		if len(lines) != 3 {
			return fmt.Errorf("want 3 error lines, got %d:\n%s", len(lines), w.run.stderr.String())
		}
		return w.stderrHas("requests[0].chats", "requests[1].id", "api_hash")
	})
	sc.Step(`^tgscout warns that the design maximum is 100 chats$`, func() error {
		return w.stderrHas("[warn]", "101 chats; the design maximum is 100")
	})
	sc.Step(`^it goes on with the start$`, func() error { return w.stdoutHas("running") })
	sc.Step(`^tgscout prints that the config has (\d+ requests?), (\d+ chats?) and the model (\S+)$`,
		func(requests, chats, model string) error {
			return w.stdoutHas(fmt.Sprintf("tgscout.yaml: %s, %s, model %s", requests, chats, model))
		})
}
