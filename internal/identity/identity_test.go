package identity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testHome(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	return h
}

func mustAdmin(t *testing.T, email string) User {
	t.Helper()
	u, err := Add(email, "", "correct horse battery", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestFirstAccountMustBeAdmin(t *testing.T) {
	testHome(t)
	if _, err := Add("user@example.com", "", "correct horse battery", RoleUser); !errors.Is(err, ErrNoAdmin) {
		t.Fatalf("first user: %v, want ErrNoAdmin", err)
	}
	if _, err := Register("user@example.com", "", "correct horse battery"); !errors.Is(err, ErrNoAdmin) {
		t.Fatalf("register before an admin: %v, want ErrNoAdmin", err)
	}
	mustAdmin(t, "admin@example.com")
	if _, err := Register("user@example.com", "", "correct horse battery"); err != nil {
		t.Fatalf("register with an admin: %v", err)
	}
	// and the first admin's role is the only way in
	if _, err := Authenticate("admin@example.com", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordIsHashedAndLoginWorks(t *testing.T) {
	h := testHome(t)
	u := mustAdmin(t, "admin@example.com")
	if u.Password == "" || !strings.HasPrefix(u.Password, "$"+"argon"+"2id"+"$v=19$") {
		t.Fatalf("password stored as %q", u.Password)
	}
	b, err := os.ReadFile(filepath.Join(h, ".config", "magpie", "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "correct horse battery") {
		t.Fatal("the password is in users.json")
	}
	if info, err := os.Stat(Path()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("users.json mode %v, %v", info.Mode(), err)
	}
	got, err := Authenticate("ADMIN@example.com", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID || got.LastLogin == nil {
		t.Fatalf("authenticated %+v", got)
	}
	if _, err := Authenticate("admin@example.com", "wrong password"); !errors.Is(err, ErrBadLogin) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, err := Authenticate("nobody@example.com", "correct horse battery"); !errors.Is(err, ErrBadLogin) {
		t.Fatalf("unknown email: %v", err)
	}
}

func TestLoginByEmailOrUsername(t *testing.T) {
	testHome(t)
	admin := mustAdmin(t, "admin@local")
	user, err := Add("test@local", "Test User", "test123456", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		login, password, id string
	}{
		{"admin", "correct horse battery", admin.ID},
		{"ADMIN", "correct horse battery", admin.ID},
		{" admin ", "correct horse battery", admin.ID},
		{"admin@local", "correct horse battery", admin.ID},
		{"admin@LOCAL", "correct horse battery", admin.ID},
		{"test", "test123456", user.ID},
		{"TEST", "test123456", user.ID},
		{"test@local", "test123456", user.ID},
	} {
		got, err := Authenticate(c.login, c.password)
		if err != nil || got.ID != c.id {
			t.Errorf("login %q: %+v, %v", c.login, got, err)
		}
	}
	if _, err := Authenticate("nobody", "test123456"); !errors.Is(err, ErrBadLogin) {
		t.Fatalf("unknown username: %v", err)
	}
	// a username shared by two accounts is not guessed: either email still
	// signs in, but the bare name is refused.
	if _, err := Add("admin@other.example", "Other", "other password", RoleUser); err != nil {
		t.Fatal(err)
	}
	if _, err := Authenticate("admin", "correct horse battery"); !errors.Is(err, ErrBadLogin) {
		t.Fatalf("ambiguous username: %v", err)
	}
	if _, err := Authenticate("admin@local", "correct horse battery"); err != nil {
		t.Fatalf("email after an ambiguous username: %v", err)
	}
	if _, err := Authenticate("admin@other.example", "other password"); err != nil {
		t.Fatalf("the other account's email: %v", err)
	}
}

func TestEmailAndPasswordPolicy(t *testing.T) {
	testHome(t)
	mustAdmin(t, "admin@example.com")
	for _, email := range []string{"", "not-an-email", "A <a@b.com>", "a b@example.com"} {
		if _, err := Register(email, "", "correct horse battery"); err == nil {
			t.Errorf("%q was taken as an email", email)
		}
	}
	for _, pw := range []string{"", "short", strings.Repeat("x", 1025)} {
		if _, err := Register("user@example.com", "", pw); err == nil {
			t.Errorf("a password of %d bytes was taken", len(pw))
		}
	}
	if _, err := Register("user@example.com", "", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if _, err := Register("USER@example.com", "", "another password"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate email, case apart: %v", err)
	}
}

func TestDisabledAccountCannotSignIn(t *testing.T) {
	testHome(t)
	u := mustAdmin(t, "admin@example.com")
	mustAdmin(t, "second@example.com") // the admin being disabled is not the last
	token, _, err := NewSession(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SetDisabled(u.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := Authenticate("admin@example.com", "correct horse battery"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled login: %v", err)
	}
	if _, err := SessionUser(token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("disabled account's session: %v", err)
	}
	if _, err := SetDisabled(u.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Authenticate("admin@example.com", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
}

func TestLastAdminIsProtected(t *testing.T) {
	testHome(t)
	admin := mustAdmin(t, "admin@example.com")
	if err := Remove(admin.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("remove: %v", err)
	}
	if _, err := SetDisabled(admin.ID, true); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("disable: %v", err)
	}
	if _, err := SetRole(admin.ID, RoleUser); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demote: %v", err)
	}
	second := mustAdmin(t, "second@example.com")
	if _, err := SetRole(admin.ID, RoleUser); err != nil {
		t.Fatalf("demote with another admin: %v", err)
	}
	// second is the last enabled admin now
	if _, err := SetDisabled(second.ID, true); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("disable the only enabled admin: %v", err)
	}
	if _, err := SetRole(admin.ID, RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := SetDisabled(second.ID, true); err != nil {
		t.Fatalf("disable with another admin on: %v", err)
	}
	if _, err := SetDisabled(admin.ID, true); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("disable the new last admin: %v", err)
	}
}

func TestPasswordResetEndsSessions(t *testing.T) {
	testHome(t)
	u := mustAdmin(t, "admin@example.com")
	token, _, err := NewSession(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetPassword(u.ID, "another password"); err != nil {
		t.Fatal(err)
	}
	if _, err := SessionUser(token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("session after a reset: %v", err)
	}
	if _, err := Authenticate("admin@example.com", "correct horse battery"); !errors.Is(err, ErrBadLogin) {
		t.Fatal("the old password still signs in")
	}
	if _, err := Authenticate("admin@example.com", "another password"); err != nil {
		t.Fatal(err)
	}
}

func TestSessionExpiryAndStorage(t *testing.T) {
	h := testHome(t)
	u := mustAdmin(t, "admin@example.com")
	now := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	clock = func() time.Time { return now }
	t.Cleanup(func() { clock = time.Now })

	token, expires, err := NewSession(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := now.Add(SessionTTL); !expires.Equal(want) {
		t.Fatalf("expires %v, want %v", expires, want)
	}
	b, err := os.ReadFile(filepath.Join(h, ".config", "magpie", "sessions.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), token) {
		t.Fatal("the session token is in sessions.json")
	}
	if !strings.Contains(string(b), tokenHash(token)) {
		t.Fatal("the session's hash isn't in sessions.json")
	}
	if info, err := os.Stat(SessionsPath()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("sessions.json mode %v, %v", info.Mode(), err)
	}
	if _, err := SessionUser(token); err != nil {
		t.Fatal(err)
	}
	clock = func() time.Time { return now.Add(SessionTTL) }
	if _, err := SessionUser(token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("expired session: %v", err)
	}
	var kept map[string]session
	b, _ = os.ReadFile(SessionsPath())
	if err := json.Unmarshal(b, &kept); err != nil {
		t.Fatal(err)
	}
	if len(kept) != 0 {
		t.Fatalf("expired session kept: %v", kept)
	}
}

func TestMaxSessionsEndsTheOldest(t *testing.T) {
	testHome(t)
	u := mustAdmin(t, "admin@example.com")
	now := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	clock = func() time.Time { return now }
	t.Cleanup(func() { clock = time.Now })
	var tokens []string
	for i := 0; i < MaxSessions+1; i++ {
		token, _, err := NewSession(u.ID)
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, token)
		now = now.Add(time.Second)
	}
	if _, err := SessionUser(tokens[0]); !errors.Is(err, ErrNoSession) {
		t.Fatal("the oldest session still signs in")
	}
	if _, err := SessionUser(tokens[len(tokens)-1]); err != nil {
		t.Fatal("the newest session doesn't sign in")
	}
}

func TestRemoveEndsSessions(t *testing.T) {
	testHome(t)
	u := mustAdmin(t, "admin@example.com")
	second := mustAdmin(t, "second@example.com")
	token, _, err := NewSession(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := ByID(second.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("the removed account is still here")
	}
	if _, err := SessionUser(token); !errors.Is(err, ErrNoSession) {
		t.Fatal("the removed account's session still signs in")
	}
	if _, err := Authenticate("second@example.com", "correct horse battery"); !errors.Is(err, ErrBadLogin) {
		t.Fatal("the removed account still signs in")
	}
	_ = u
}

func TestChangingRoleKeepsTheAccount(t *testing.T) {
	testHome(t)
	admin := mustAdmin(t, "admin@example.com")
	user, err := Register("user@example.com", "Ann", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != RoleUser || user.Name != "Ann" {
		t.Fatalf("registered %+v", user)
	}
	if _, err := SetRole(user.ID, RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if got, _ := ByID(user.ID); got.Role != RoleAdmin || got.Password != user.Password {
		t.Fatalf("after promotion %+v", got)
	}
	if _, err := SetRole(user.ID, RoleUser); err != nil {
		t.Fatal(err)
	}
	_ = admin
}
