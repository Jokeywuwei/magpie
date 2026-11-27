// Package identity holds the platform's accounts and their web sessions:
// who may sign in, whether they are a user or an admin, and how a browser
// is known between requests. Gateway API keys are internal/access's; an
// account's web sign-in is not a gateway key, and the two never stand in
// for each other.
//
// Users are kept in users.json beside magpie's settings, readable by this
// account alone, with each password an argon2id hash: the password itself
// is never written, sent or logged. Sessions are kept in sessions.json as
// the SHA-256 of their tokens, so a token read from that file still can't
// be used.
package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/settings"
	"golang.org/x/crypto/argon2"
)

// Role is what an account may do: use the user pages and its own keys, or
// everything, the admin pages and every account's keys among them.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// ParseRole reads a role as it is kept: user or admin.
func ParseRole(v string) (Role, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "user", "member":
		return RoleUser, nil
	case "admin", "administrator":
		return RoleAdmin, nil
	}
	return "", fmt.Errorf("role is user or admin, not %q", v)
}

// Errors callers tell apart; everything else is a validation failure.
var (
	ErrNotFound   = errors.New("no account with that email")
	ErrEmailTaken = errors.New("an account with that email already exists")
	ErrBadLogin   = errors.New("email, username or password is not right")
	ErrDisabled   = errors.New("this account is disabled")
	ErrNoAdmin    = errors.New("no admin exists yet: create one with `magpie user add <email> --role admin`")
	ErrLastAdmin  = errors.New("the last admin can't be removed, disabled or made a user")
	ErrNoSession  = errors.New("not signed in")
)

// User is one account as it is stored. Password is the argon2id hash in
// its PHC form; Account, never User, is what a page is sent.
type User struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      Role       `json:"role"`
	Password  string     `json:"password"`
	Disabled  bool       `json:"disabled,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	LastLogin *time.Time `json:"lastLoginAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// Account is a user without the hash: what a page, the CLI's list and the
// admin API may see.
type Account struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      Role       `json:"role"`
	Disabled  bool       `json:"disabled,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	LastLogin *time.Time `json:"lastLoginAt,omitempty"`
}

// Account is u without what signs it in.
func (u User) Account() Account {
	return Account{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, Disabled: u.Disabled, CreatedAt: u.CreatedAt, LastLogin: u.LastLogin}
}

// Path is the accounts' file, beside magpie's own settings.
func Path() string { return filepath.Join(settings.Dir(), "users.json") }

// SessionsPath is where the hashes of the sign-in tokens are kept.
func SessionsPath() string { return filepath.Join(settings.Dir(), "sessions.json") }

// clock is time.Now, for tests to move.
var clock = time.Now

// mu is held for every read a change is made from, and the write, across
// magpies too: a CLI add and a page's sign-up never write over each other.
// sessions.json is guarded by the same lock, so a disabled account's
// sessions and the account are one change.
var mu = filelock.Mutex{Path: func() string { return Path() + ".lock" }}

func load() ([]User, error) {
	b, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return []User{}, nil
	}
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(b, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func save(users []User) error {
	b, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(Path(), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(Path(), 0o600); err != nil {
		return err
	}
	return edit.WriteAtomic(Path(), append(b, '\n'))
}

// cleanEmail is email normalized: lower case, no display name, one @.
func cleanEmail(email string) (string, error) {
	e := strings.TrimSpace(email)
	if e == "" || len(e) > 254 {
		return "", errors.New("enter an email address")
	}
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Name != "" || addr.Address != e {
		return "", fmt.Errorf("%q is not an email address", e)
	}
	return strings.ToLower(e), nil
}

func cleanName(name, email string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		n = email[:strings.IndexByte(email, '@')]
	}
	if utf8.RuneCountInString(n) > 80 {
		return "", errors.New("a name is 80 characters at most")
	}
	return n, nil
}

func cleanPassword(password string) error {
	if len(password) < 8 {
		return errors.New("a password is 8 characters at least")
	}
	if len(password) > 1024 {
		return errors.New("a password is 1024 characters at most")
	}
	return nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Argon2id's parameters, OWASP's least: 19 MiB, two passes, one lane. A
// sign-in takes some tens of milliseconds.
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

func b64(b []byte) string { return base64.RawStdEncoding.EncodeToString(b) }

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return "$" + "argon" + "2id" + fmt.Sprintf("$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads, b64(salt), b64(key)), nil
}

// checkPassword tells whether password is phc's. A hash that doesn't parse
// is never a match; the compare is constant time.
func checkPassword(phc, password string) bool {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory, passes uint32
	var lanes uint8
	if n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &passes, &lanes); n != 3 || err != nil {
		return false
	}
	if memory == 0 || memory > 1<<20 || passes == 0 || passes > 16 || lanes == 0 || lanes > 32 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) < 16 || len(want) > 128 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, passes, memory, lanes, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummy is a hash of a password no one knows, checked when a login has no
// account so that a missing account and a wrong password take the same work.
var dummy = sync.OnceValue(func() string {
	h, err := hashPassword(newRandomString(24))
	if err != nil {
		return ""
	}
	return h
})

func newRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func indexEmail(users []User, email string) int {
	return slices.IndexFunc(users, func(u User) bool { return u.Email == email })
}

// indexLogin finds the account a sign-in names: an email, or the part of an
// email before @. A username that names more than one account is not guessed;
// the caller is told the sign-in failed and must use the email. Emails are
// kept lower case, and usernames are compared the same way.
func indexLogin(users []User, login string) int {
	l := strings.ToLower(strings.TrimSpace(login))
	if l == "" {
		return -1
	}
	if strings.Contains(l, "@") {
		return indexEmail(users, l)
	}
	found := -1
	for i := range users {
		at := strings.IndexByte(users[i].Email, '@')
		if at < 0 || !strings.EqualFold(users[i].Email[:at], l) {
			continue
		}
		if found >= 0 {
			return -1
		}
		found = i
	}
	return found
}

func indexID(users []User, id string) int {
	return slices.IndexFunc(users, func(u User) bool { return u.ID == id })
}

func enabledAdmins(users []User) int {
	n := 0
	for _, u := range users {
		if u.Role == RoleAdmin && !u.Disabled {
			n++
		}
	}
	return n
}

// Add creates an account. The first account a platform has must be an
// admin, so a platform never starts with users and no one to manage it.
func Add(email, name, password string, role Role) (User, error) {
	e, err := cleanEmail(email)
	if err != nil {
		return User{}, err
	}
	n, err := cleanName(name, e)
	if err != nil {
		return User{}, err
	}
	if err := cleanPassword(password); err != nil {
		return User{}, err
	}
	r, err := ParseRole(string(role))
	if err != nil {
		return User{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return User{}, err
	}
	if indexEmail(users, e) >= 0 {
		return User{}, ErrEmailTaken
	}
	if len(users) == 0 && r != RoleAdmin {
		return User{}, ErrNoAdmin
	}
	id, err := newID()
	if err != nil {
		return User{}, err
	}
	now := clock().UTC()
	u := User{ID: id, Email: e, Name: n, Role: r, Password: hash, CreatedAt: now, UpdatedAt: now}
	if err := save(append(users, u)); err != nil {
		return User{}, err
	}
	return u, nil
}

// Register creates a user's own account. It is open only once an admin
// exists: the first admin is made with `magpie user add --role admin`.
func Register(email, name, password string) (User, error) {
	e, err := cleanEmail(email)
	if err != nil {
		return User{}, err
	}
	n, err := cleanName(name, e)
	if err != nil {
		return User{}, err
	}
	if err := cleanPassword(password); err != nil {
		return User{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return User{}, err
	}
	if enabledAdmins(users) == 0 {
		return User{}, ErrNoAdmin
	}
	if indexEmail(users, e) >= 0 {
		return User{}, ErrEmailTaken
	}
	id, err := newID()
	if err != nil {
		return User{}, err
	}
	now := clock().UTC()
	u := User{ID: id, Email: e, Name: n, Role: RoleUser, Password: hash, CreatedAt: now, UpdatedAt: now}
	if err := save(append(users, u)); err != nil {
		return User{}, err
	}
	return u, nil
}

// Authenticate checks an email or username and a password, and keeps the
// time it did. The username is the part of the email before @.
func Authenticate(login, password string) (User, error) {
	mu.Lock()
	users, err := load()
	var u User
	i := indexLogin(users, login)
	if err == nil && i >= 0 {
		u = users[i]
	}
	mu.Unlock()
	if err != nil {
		return User{}, err
	}
	if i < 0 {
		checkPassword(dummy(), password)
		return User{}, ErrBadLogin
	}
	if !checkPassword(u.Password, password) {
		return User{}, ErrBadLogin
	}
	if u.Disabled {
		return User{}, ErrDisabled
	}
	// the account may have changed while the hash was checked
	mu.Lock()
	defer mu.Unlock()
	users, err = load()
	if err != nil {
		return User{}, err
	}
	i = indexID(users, u.ID)
	if i < 0 || users[i].Disabled || users[i].Password != u.Password {
		return User{}, ErrBadLogin
	}
	now := clock().UTC()
	users[i].LastLogin = &now
	users[i].UpdatedAt = now
	if err := save(users); err != nil {
		return User{}, err
	}
	return users[i], nil
}

// ByID is the account with id.
func ByID(id string) (User, error) {
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return User{}, err
	}
	if i := indexID(users, id); i >= 0 {
		return users[i], nil
	}
	return User{}, ErrNotFound
}

// ByEmail is the account with email, as it is kept.
func ByEmail(email string) (User, error) {
	e, err := cleanEmail(email)
	if err != nil {
		return User{}, ErrNotFound
	}
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return User{}, err
	}
	if i := indexEmail(users, e); i >= 0 {
		return users[i], nil
	}
	return User{}, ErrNotFound
}

// List is every account, without a hash.
func List() ([]Account, error) {
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(users))
	for i, u := range users {
		out[i] = u.Account()
	}
	return out, nil
}

// SetRole makes an admin a user or a user an admin. The last enabled admin
// is never made a user.
func SetRole(id string, role Role) (User, error) {
	r, err := ParseRole(string(role))
	if err != nil {
		return User{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return User{}, err
	}
	i := indexID(users, id)
	if i < 0 {
		return User{}, ErrNotFound
	}
	if users[i].Role == r {
		return users[i], nil
	}
	if users[i].Role == RoleAdmin && r != RoleAdmin && enabledAdmins(users) <= 1 {
		return User{}, ErrLastAdmin
	}
	users[i].Role = r
	users[i].UpdatedAt = clock().UTC()
	// a role is what an existing session carries: end them before the new
	// role is written, so a browser signed in as a user never becomes an
	// admin by the promotion alone. Ending first leaves a lost session, not
	// a widened one, if the accounts can't be written.
	if err := endUserSessionsLocked(id); err != nil {
		return User{}, err
	}
	if err := save(users); err != nil {
		return User{}, err
	}
	return users[i], nil
}

// SetDisabled turns an account off or on again. A disabled account cannot
// sign in, and its sessions end. The last enabled admin is never disabled.
func SetDisabled(id string, disabled bool) (User, error) {
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return User{}, err
	}
	i := indexID(users, id)
	if i < 0 {
		return User{}, ErrNotFound
	}
	if users[i].Disabled == disabled {
		return users[i], nil
	}
	if disabled && users[i].Role == RoleAdmin && !users[i].Disabled && enabledAdmins(users) <= 1 {
		return User{}, ErrLastAdmin
	}
	users[i].Disabled = disabled
	users[i].UpdatedAt = clock().UTC()
	if err := save(users); err != nil {
		return User{}, err
	}
	if disabled {
		if err := endUserSessionsLocked(id); err != nil {
			return User{}, err
		}
	}
	return users[i], nil
}

// SetPassword replaces an account's password and ends every session it
// had: a reset signs the other browsers out.
func SetPassword(id, password string) error {
	if err := cleanPassword(password); err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return err
	}
	i := indexID(users, id)
	if i < 0 {
		return ErrNotFound
	}
	users[i].Password = hash
	users[i].UpdatedAt = clock().UTC()
	// a session isn't tied to the password it signed in with, so end them
	// before the new hash is written: a failure can lose a session, never
	// leave one alive on the old password
	if err := endUserSessionsLocked(id); err != nil {
		return err
	}
	return save(users)
}

// Remove deletes an account and ends its sessions. The last enabled admin
// is never removed.
func Remove(id string) error {
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return err
	}
	i := indexID(users, id)
	if i < 0 {
		return ErrNotFound
	}
	if users[i].Role == RoleAdmin && !users[i].Disabled && enabledAdmins(users) <= 1 {
		return ErrLastAdmin
	}
	users = slices.Delete(users, i, i+1)
	if err := save(users); err != nil {
		return err
	}
	return endUserSessionsLocked(id)
}
