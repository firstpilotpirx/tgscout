package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDotEnvReadsKeysCommentsAndQuotes(t *testing.T) {
	values := parseDotEnv("# Telegram\nTG_API_ID=12345\n\nTG_API_HASH=\"made-up-hash\"\nTG_SESSION_FILE='./my.session'\nbroken line\n")

	assert.Equal(t, map[string]string{"TG_API_ID": "12345", "TG_API_HASH": "made-up-hash",
		"TG_SESSION_FILE": "./my.session"}, values)
}

func TestCredentialsPreferTheEnvironment(t *testing.T) {
	env := map[string]string{"TG_API_HASH": "from-env"}
	creds, err := loadCredentials(lookupIn(env), map[string]string{"TG_API_ID": "12345", "TG_API_HASH": "from-file"},
		"./conf/tgscout.yaml")

	require.NoError(t, err)
	assert.Equal(t, 12345, creds.apiID)
	assert.Equal(t, "from-env", creds.apiHash)
	assert.Equal(t, filepath.Join("conf", "tgscout.session"), creds.sessionFile, "next to the config")
}

func TestCredentialsNameTheMissingVariable(t *testing.T) {
	_, err := loadCredentials(lookupIn(nil), map[string]string{"TG_API_ID": "12345"}, "./tgscout.yaml")
	assert.EqualError(t, err, "TG_API_HASH is not set")

	_, err = loadCredentials(lookupIn(nil), map[string]string{"TG_API_HASH": "x"}, "./tgscout.yaml")
	assert.EqualError(t, err, "TG_API_ID is not set")

	_, err = loadCredentials(lookupIn(nil), map[string]string{"TG_API_ID": "abc", "TG_API_HASH": "x"}, "./tgscout.yaml")
	assert.EqualError(t, err, "TG_API_ID is not a number")
}

func TestCredentialsTakeTheSessionFileGiven(t *testing.T) {
	creds, err := loadCredentials(lookupIn(map[string]string{"TG_API_ID": "1", "TG_API_HASH": "x",
		"TG_SESSION_FILE": "/tmp/made-up.session"}), nil, "./tgscout.yaml")

	require.NoError(t, err)
	assert.Equal(t, "/tmp/made-up.session", creds.sessionFile)
}

func lookupIn(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
