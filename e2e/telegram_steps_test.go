package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

const (
	twoJoinedChats = "owner: '@made_up_owner'\nlm_studio:\n  model: example-model\n" +
		"requests:\n  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market', '@made_up_river']\n"
	threeChats = "owner: '@made_up_owner'\nlm_studio:\n  model: example-model\n" +
		"requests:\n  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market', '@made_up_river', '@made_up_gone']\n"
	numericOwner = "owner: '100200300'\nlm_studio:\n  model: example-model\n" +
		"requests:\n  - id: bike\n    find: a kids bike\n    chats: ['@made_up_market']\n"
)

func telegramSteps(sc *godog.ScenarioContext, w *world) {
	telegramGivenSteps(sc, w)
	telegramWhenSteps(sc, w)
	telegramThenSteps(sc, w)
	lookupThenSteps(sc, w)
}

func telegramGivenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^an api id but no api hash$`, func() { w.tg.noAPIHash = true })
	sc.Step(`^no session file yet and an account with a two-step password$`, func() {
		w.tg.session, w.tg.fake.SessionValid, w.tg.fake.Password = false, false, madeUpPassword
	})
	sc.Step(`^a valid private session file$`, func() {})
	sc.Step(`^a session file readable by other users$`, func() { w.tg.sessionMode = 0o644 })
	sc.Step(`^a private session file that Telegram no longer accepts$`, func() { w.tg.fake.SessionValid = false })
	sc.Step(`^a config whose 2 chats the account has joined and whose owner Telegram knows$`, func() error {
		return w.writeConfig(twoJoinedChats)
	})
	sc.Step(`^a config where one chat the account has not joined and one chat does not exist$`, func() error {
		w.tg.fake.Chats = map[string]string{"@made_up_market": "joined", "@made_up_river": "not_joined"}
		return w.writeConfig(threeChats)
	})
	sc.Step(`^Telegram asks to wait 1 second before the lookup$`, func() { w.tg.fake.FloodWaitSeconds = 1 })
	sc.Step(`^a config whose owner is a numeric ID that is not among the account's contacts$`, func() error {
		return w.writeConfig(numericOwner)
	})
}

func telegramWhenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^the owner starts tgscout with its input not coming from a terminal$`, func() error { return w.start() })
	sc.Step(`^the owner starts tgscout on a terminal and types the phone number, the login code and the password$`,
		func() error { w.echo = true; return w.signInOnTerminal(madeUpCode) })
	sc.Step(`^the owner starts tgscout on a terminal and types a wrong login code$`, func() error {
		if err := w.startTerminal(); err != nil {
			return err
		}
		return w.typeAll([]string{"Phone number: ", madeUpPhone}, []string{"Login code: ", madeUpWrongCode})
	})
	sc.Step(`^the owner starts tgscout on a terminal and types a wrong code, then the right code and the password$`,
		func() error { return w.signInOnTerminal(madeUpWrongCode, madeUpCode) })
}

func telegramThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^tgscout prints an error that names the missing api hash$`, func() error {
		return w.stderrHas("[fail] TG_API_HASH is not set", ".env.example")
	})
	sc.Step(`^tgscout says it signed in as the account's name$`, func() error {
		return w.outputHas("[ok] signed in as " + madeUpAccount)
	})
	sc.Step(`^the session file exists and only its owner can read or write it$`, w.sessionIsPrivate)
	sc.Step(`^the password is not shown while typed$`, w.passwordHidden)
	sc.Step(`^tgscout says the code is wrong and asks for the code again$`, func() error {
		if err := w.outputHas("[fail] the code is wrong — try again"); err != nil {
			return err
		}
		return w.run.waitFor("Login code: ", 2)
	})
	sc.Step(`^no session file is saved$`, w.noSessionFile)
	sc.Step(`^tgscout prints an error that says to sign in once from a terminal$`, func() error {
		if strings.Contains(w.run.output(), "Phone number") {
			return fmt.Errorf("tgscout asked for input:\n%s", w.run.output())
		}
		return w.stderrHas("[fail] first sign-in needs a terminal", "sign in once from a terminal first")
	})
	sc.Step(`^tgscout says it signed in from the saved session without asking anything$`, func() error {
		if strings.Contains(w.run.output(), "Phone number") {
			return fmt.Errorf("tgscout asked for input:\n%s", w.run.output())
		}
		return w.stdoutHas("[ok] signed in as " + madeUpAccount + " (saved session)")
	})
	sc.Step(`^it goes on running$`, func() error { return w.stdoutHas("[run] running") })
	sc.Step(`^it watches 1 chat$`, func() error { return w.stdoutHas("[ok] watching 1 chat", "[run] running") })
	sc.Step(`^tgscout prints an error that says how to make the session file private$`, func() error {
		path := filepath.Join(w.dir, sessionFileName)
		return w.stderrHas("[fail] the session file is open to other users", "mode 644", "chmod 600 "+path)
	})
	sc.Step(`^tgscout prints an error that tells the owner to sign in again$`, func() error {
		return w.stderrHas("[fail] Telegram no longer accepts the session", "then start tgscout from a terminal")
	})
}

func lookupThenSteps(sc *godog.ScenarioContext, w *world) {
	sc.Step(`^tgscout says it found the owner and watches 2 chats$`, func() error {
		return w.stdoutHas("[ok] owner found: @made_up_owner", "[ok] watching 2 chats")
	})
	sc.Step(`^tgscout prints an error that lists both chats, each with its reason$`, func() error {
		return w.stderrHas("[fail] 2 chats not found", "@made_up_river  the account has not joined it",
			"@made_up_gone   no such chat", "join them with tgscout's account")
	})
	sc.Step(`^tgscout prints that it waits 1s$`, func() error {
		return w.stderrHas("[warn] Telegram asks to wait 1s (FLOOD_WAIT), then goes on")
	})
	sc.Step(`^tgscout prints an error that says to add the owner to the account's contacts$`, func() error {
		return w.stderrHas("[fail] owner not found: 100200300", "add the owner to the contacts of tgscout's account")
	})
	sc.Step(`^the output holds no phone number, api hash, login code, password or session data$`, w.noSecrets)
}

// startTerminal runs tgscout on a pseudo-terminal with the scenario's config.
func (w *world) startTerminal() error {
	env, err := w.tg.write(w.dir)
	if err != nil {
		return err
	}
	w.run, err = startOnTerminal(w.dir, env, w.echo, "--config", "./"+w.config)
	return err
}

// signInOnTerminal types the phone, each code in turn and the password, then waits for the run.
func (w *world) signInOnTerminal(codes ...string) error {
	if err := w.startTerminal(); err != nil {
		return err
	}
	answers := [][]string{{"Phone number: ", madeUpPhone}}
	for _, code := range codes {
		answers = append(answers, []string{"Login code: ", code})
	}
	if err := w.typeAll(answers...); err != nil {
		return err
	}
	if err := w.run.waitFor("Two-step password: ", 1); err != nil {
		return err
	}
	if err := w.run.waitEchoOff(); err != nil {
		return err
	}
	if err := w.run.typeAfter("Two-step password: ", 1, madeUpPassword); err != nil {
		return err
	}
	return w.run.settle()
}

// typeAll answers each prompt in turn; a prompt asked again counts once more.
func (w *world) typeAll(answers ...[]string) error {
	asked := map[string]int{}
	for _, answer := range answers {
		asked[answer[0]]++
		if err := w.run.typeAfter(answer[0], asked[answer[0]], answer[1]); err != nil {
			return err
		}
	}
	return nil
}

func (w *world) outputHas(want ...string) error { return has("output", w.run.output(), want) }

func (w *world) sessionIsPrivate() error {
	info, err := os.Stat(filepath.Join(w.dir, sessionFileName))
	if err != nil {
		return fmt.Errorf("session file: %w", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		return fmt.Errorf("session file mode %o, want 600", mode)
	}
	return nil
}

func (w *world) noSessionFile() error {
	if _, err := os.Stat(filepath.Join(w.dir, sessionFileName)); !os.IsNotExist(err) {
		return fmt.Errorf("a session file exists: %v", err)
	}
	return nil
}

func (w *world) noSecrets() error {
	output := w.run.output()
	if err := w.outputHas("[ok] signed in as "+madeUpAccount, "[run] running"); err != nil {
		return err
	}
	for _, secret := range []string{madeUpPhone, madeUpAPIHash, madeUpCode, madeUpWrongCode, madeUpPassword, madeUpSession} {
		if strings.Contains(output, secret) {
			return fmt.Errorf("the output holds the secret %q:\n%s", secret, output)
		}
	}
	return nil
}

// passwordHidden: the terminal showed the typed phone, so echo was on, but not the password.
func (w *world) passwordHidden() error {
	output := w.run.output()
	if !strings.Contains(output, madeUpPhone) {
		return fmt.Errorf("the terminal did not echo the phone, so the check proves nothing:\n%s", output)
	}
	if strings.Contains(output, madeUpPassword) {
		return fmt.Errorf("the password was shown while typed:\n%s", output)
	}
	return nil
}
