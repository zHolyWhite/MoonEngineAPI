package auth

import (
	"crypto/subtle"
	"strings"
)

type entry struct {
	name string
	key  []byte
}

type Authenticator struct {
	entries []entry
}

func New(keys map[string]string) *Authenticator {
	a := &Authenticator{entries: make([]entry, 0, len(keys))}
	for name, key := range keys {
		a.entries = append(a.entries, entry{name: name, key: []byte(key)})
	}
	return a
}

func (a *Authenticator) Authenticate(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := []byte(strings.TrimSpace(header[len(prefix):]))
	found := ""
	ok := false
	for _, e := range a.entries {
		if subtle.ConstantTimeCompare(token, e.key) == 1 {
			found = e.name
			ok = true
		}
	}
	return found, ok
}