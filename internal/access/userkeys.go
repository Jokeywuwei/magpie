// A platform user's own API keys: made in the console, kept hashed, and
// owned by one account. The gateway's older keys are this package's own,
// with their secret kept so an administrator can copy it again; a user's
// key is never kept that way. Only the prefix is shown after it is made.
package access

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yetone/magpie/internal/modelaccess"
)

// MaxUserKeys is how many keys one account may keep. An account that needs
// more is a platform's business, not a runaway list.
const MaxUserKeys = 50

// ErrBadModels is a grant naming a model the platform hasn't opened; the
// console answers it as a bad request, not as a storage failure.
var ErrBadModels = errors.New("a model the platform hasn't opened")

// ErrEmptyGrant is an explicit grant with no models: a user key must name
// at least one open model, and an ownerless key's empty list is not this.
var ErrEmptyGrant = errors.New("choose at least one open model for this key")

type badModels struct{ model string }

func (e badModels) Error() string        { return e.model + " isn't open to platform users now" }
func (e badModels) Is(target error) bool { return target == ErrBadModels }

// UserKey is one of an account's keys as its console shows it: never the
// hash, and never the secret.
type UserKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Off       bool      `json:"off"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  time.Time `json:"lastUsed,omitzero"`
	Models    []string  `json:"models,omitempty"`
	// LegacyModels says the key's models were filled once from the open
	// catalog when model access arrived, rather than chosen in the console.
	LegacyModels bool `json:"legacyModels,omitempty"`
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// prefixOf is the part of a key a list may show: the prefix and the first
// eight characters after it, enough to tell two keys apart.
func prefixOf(secret string) string {
	prefix := Prefix
	if strings.HasPrefix(secret, UserPrefix) {
		prefix = UserPrefix
	}
	n := len(prefix) + 8
	if n > len(secret) {
		n = len(secret)
	}
	return secret[:n]
}

func (k Key) userKey() UserKey {
	return UserKey{ID: k.ID, Name: k.Name, Prefix: k.Prefix, Off: k.Off, CreatedAt: k.CreatedAt, LastUsed: k.LastUsed, Models: k.Models, LegacyModels: k.LegacyModels}
}

func cleanUserKeyName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" || utf8.RuneCountInString(n) > 120 {
		return "", errors.New("Use a name between 1 and 120 characters")
	}
	return n, nil
}

// UserKeys lists owner's keys, newest first. Another account's keys are
// not in the answer at all.
func UserKeys(owner string) ([]UserKey, error) {
	if owner == "" {
		return []UserKey{}, nil
	}
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return nil, err
	}
	if fillLegacyModels(keys) {
		if err := save(keys); err != nil {
			return nil, err
		}
	}
	out := make([]UserKey, 0, len(keys))
	for _, k := range keys {
		if k.Owner == owner {
			out = append(out, k.userKey())
		}
	}
	slices.SortFunc(out, func(a, b UserKey) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

// cleanUserKeyModels is a key's model grant: exact ids of models the
// platform opens now, never patterns. nil asks for every open model, the
// console's default; an explicit empty list is refused, never read as
// every model, because an empty owner-held key was a cross-tenant hole.
func cleanUserKeyModels(models []string) ([]string, error) {
	if models == nil {
		return modelaccess.OpenIDs(), nil
	}
	open := modelaccess.OpenIDs()
	ok := make(map[string]bool, len(open))
	for _, id := range open {
		ok[id] = true
	}
	out := make([]string, 0, len(models))
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" || slices.Contains(out, m) {
			continue
		}
		if !ok[m] {
			return nil, badModels{m}
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil, ErrEmptyGrant
	}
	return out, nil
}

// fillLegacyModels gives a user key made before model access an explicit
// grant: the models open when it is first read. A key that already named
// models keeps them. Magpie's own and the administrator's keys are left
// alone. It reports whether anything changed, for the caller to save.
func fillLegacyModels(keys []Key) bool {
	var open []string
	openRead := false
	changed := false
	for i := range keys {
		k := &keys[i]
		if k.Owner == "" || k.ModelsSet {
			continue
		}
		if len(k.Models) == 0 {
			if !openRead {
				open = modelaccess.OpenIDs()
				openRead = true
			}
			k.Models = open
			k.LegacyModels = true
		}
		k.ModelsSet = true
		changed = true
	}
	return changed
}

// CreateUserKey makes a key owned by owner, and returns the full secret
// once: only its hash is kept, so this is the one time it can be shown.
// models is the grant; nil means every model open now.
func CreateUserKey(owner, name string, models []string) (UserKey, string, error) {
	if owner == "" {
		return UserKey{}, "", errors.New("an account is needed for a key")
	}
	n, err := cleanUserKeyName(name)
	if err != nil {
		return UserKey{}, "", err
	}
	grant, err := cleanUserKeyModels(models)
	if err != nil {
		return UserKey{}, "", err
	}
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return UserKey{}, "", err
	}
	mine := 0
	for _, k := range keys {
		if k.Owner == owner {
			mine++
		}
	}
	if mine >= MaxUserKeys {
		return UserKey{}, "", fmt.Errorf("this account has %d API keys already: remove one first", MaxUserKeys)
	}
	id, err := random(12)
	if err != nil {
		return UserKey{}, "", err
	}
	token, err := random(24)
	if err != nil {
		return UserKey{}, "", err
	}
	secret := UserPrefix + token
	now := Now().Round(0)
	k := Key{ID: id, Name: n, Owner: owner, Hash: hashSecret(secret), Prefix: prefixOf(secret), Models: grant, ModelsSet: true, MCP: []string{NoMCP}, CreatedAt: now}
	keys = append(keys, k)
	if err := save(keys); err != nil {
		return UserKey{}, "", err
	}
	return k.userKey(), secret, nil
}

// UpdateUserKey changes one of owner's keys in one write: off when off is
// given, its model grant when models is. A key that is not owner's is not
// found.
func UpdateUserKey(owner, id string, off *bool, models *[]string) (UserKey, error) {
	var grant []string
	var err error
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return UserKey{}, err
	}
	i := slices.IndexFunc(keys, func(k Key) bool { return k.ID == id && k.Owner != "" && k.Owner == owner })
	if i < 0 {
		return UserKey{}, ErrNoKey
	}
	if models != nil {
		if grant, err = cleanUserKeyModels(*models); err != nil {
			return UserKey{}, err
		}
	}
	if off != nil {
		keys[i].Off = *off
	}
	if models != nil {
		keys[i].Models, keys[i].ModelsSet, keys[i].LegacyModels = grant, true, false
	}
	if err := save(keys); err != nil {
		return UserKey{}, err
	}
	return keys[i].userKey(), nil
}

// SetUserKeyModels replaces one of owner's keys' model grant: exact ids
// of models open now. A key that is not owner's is not found.
func SetUserKeyModels(owner, id string, models []string) (UserKey, error) {
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return UserKey{}, err
	}
	i := slices.IndexFunc(keys, func(k Key) bool { return k.ID == id && k.Owner != "" && k.Owner == owner })
	if i < 0 {
		return UserKey{}, ErrNoKey
	}
	grant, err := cleanUserKeyModels(models)
	if err != nil {
		return UserKey{}, err
	}
	keys[i].Models, keys[i].ModelsSet, keys[i].LegacyModels = grant, true, false
	if err := save(keys); err != nil {
		return UserKey{}, err
	}
	return keys[i].userKey(), nil
}

// SetUserKeyOff turns one of owner's keys off, or on again. A key that is
// not owner's is not found, so another account's key is never touched.
func SetUserKeyOff(owner, id string, off bool) (UserKey, error) {
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return UserKey{}, err
	}
	i := slices.IndexFunc(keys, func(k Key) bool { return k.ID == id && k.Owner != "" && k.Owner == owner })
	if i < 0 {
		return UserKey{}, ErrNoKey
	}
	keys[i].Off = off
	if err := save(keys); err != nil {
		return UserKey{}, err
	}
	return keys[i].userKey(), nil
}

// DeleteUserKey removes one of owner's keys. Its next request fails at
// once: Check reads the store and finds no key.
func DeleteUserKey(owner, id string) error {
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(keys, func(k Key) bool { return k.ID == id && k.Owner != "" && k.Owner == owner })
	if i < 0 {
		return ErrNoKey
	}
	keys = slices.Delete(keys, i, i+1)
	return save(keys)
}

// DeleteUserKeys removes every key an account had, for an account removed
// from the platform. A key of another's is never touched.
func DeleteUserKeys(owner string) (int, error) {
	if owner == "" {
		return 0, nil
	}
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return 0, err
	}
	before := len(keys)
	keys = slices.DeleteFunc(keys, func(k Key) bool { return k.Owner == owner })
	if len(keys) == before {
		return 0, nil
	}
	return before - len(keys), save(keys)
}
