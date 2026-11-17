package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// The variables of tgscout's Telegram account; they live in .env, never in tgscout.yaml.
const (
	envAPIID       = "TG_API_ID"
	envAPIHash     = "TG_API_HASH"
	envSessionFile = "TG_SESSION_FILE"
	dotEnvFile     = ".env"
)

// credentials are the Telegram app keys and where the session lives.
type credentials struct {
	apiID       int
	apiHash     string
	sessionFile string
}

// parseDotEnv reads KEY=VALUE lines; # starts a comment, quotes around a value are dropped.
func parseDotEnv(text string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		values[strings.TrimSpace(key)] = value
	}
	return values
}

// loadCredentials takes each variable from the environment, else from .env;
// the session file defaults to tgscout.session next to the config.
func loadCredentials(lookup func(string) (string, bool), dotEnv map[string]string, config string) (credentials, error) {
	get := func(key string) string {
		if value, ok := lookup(key); ok && value != "" {
			return value
		}
		return dotEnv[key]
	}
	idText, hash := get(envAPIID), get(envAPIHash)
	if idText == "" {
		return credentials{}, fmt.Errorf("%s is not set", envAPIID)
	}
	id, err := strconv.Atoi(idText)
	if err != nil {
		return credentials{}, fmt.Errorf("%s is not a number", envAPIID)
	}
	if hash == "" {
		return credentials{}, fmt.Errorf("%s is not set", envAPIHash)
	}
	session := get(envSessionFile)
	if session == "" {
		session = filepath.Join(filepath.Dir(config), "tgscout.session")
	}
	return credentials{apiID: id, apiHash: hash, sessionFile: session}, nil
}
