// Package domain holds the values the matching component works with.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrBadChatRef means the text is not a chat reference.
var ErrBadChatRef = errors.New("not a chat reference: use @username, a t.me link or a numeric ID")

// ErrBadOwner means the text is not an owner reference.
var ErrBadOwner = errors.New("not an owner: use @username or a numeric ID")

var (
	usernamePattern = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9_]{4,31})$`)
	linkPattern     = regexp.MustCompile(`^(?:https://)?t\.me/([A-Za-z][A-Za-z0-9_]{4,31})$`)
	idPattern       = regexp.MustCompile(`^-?[0-9]+$`)
)

// ChatRef is how the owner names a chat: @username, a public t.me link or a
// numeric ID of a private chat. Whether the chat exists is checked at sign-in.
type ChatRef struct{ text, key string }

// NewChatRef checks the form of a chat reference.
func NewChatRef(text string) (ChatRef, error) {
	if key, ok := usernameKey(text); ok {
		return ChatRef{text: text, key: key}, nil
	}
	if match := linkPattern.FindStringSubmatch(text); match != nil {
		return ChatRef{text: text, key: "@" + strings.ToLower(match[1])}, nil
	}
	if idPattern.MatchString(text) {
		return ChatRef{text: text, key: text}, nil
	}
	return ChatRef{}, fmt.Errorf("%w: %q", ErrBadChatRef, text)
}

// String is the reference as the owner wrote it.
func (r ChatRef) String() string { return r.text }

// Key is the same for every reference to one chat: a link and @username of
// one username give one key, whatever the letter case.
func (r ChatRef) Key() string { return r.key }

// Owner is the Telegram user tgscout writes to: @username or a numeric ID.
type Owner struct{ text string }

// NewOwner checks the form of an owner reference.
func NewOwner(text string) (Owner, error) {
	if _, ok := usernameKey(text); ok || idPattern.MatchString(text) {
		return Owner{text: text}, nil
	}
	return Owner{}, fmt.Errorf("%w: %q", ErrBadOwner, text)
}

// String is the owner as written in the config.
func (o Owner) String() string { return o.text }

func usernameKey(text string) (string, bool) {
	match := usernamePattern.FindStringSubmatch(text)
	if match == nil {
		return "", false
	}
	return "@" + strings.ToLower(match[1]), true
}
