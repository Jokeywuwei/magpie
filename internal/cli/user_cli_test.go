package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/identity"
)

// runWithStdin runs magpie with stdin holding in, for the password an
// account command reads there.
func runWithStdin(t *testing.T, in string, args ...string) (string, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(in); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = f
	defer func() {
		os.Stdin = old
		f.Close()
	}()
	return runOut(t, args...)
}

func TestUserCLI(t *testing.T) {
	h := helpHome(t)

	out, err := runOut(t, "user", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty list: %q", out)
	}
	// the first account must be an admin
	if _, err := runWithStdin(t, "correct horse battery\n", "user", "add", "user@example.com", "--password-stdin"); err == nil || !strings.Contains(err.Error(), "admin") {
		t.Fatalf("first user: %v", err)
	}
	out, err = runWithStdin(t, "correct horse battery\n", "user", "add", "admin@example.com", "--name", "Admin", "--role", "admin", "--password-stdin")
	if err != nil || !strings.Contains(out, "admin@example.com") {
		t.Fatalf("add admin: %q %v", out, err)
	}
	usersPath := filepath.Join(h, ".config", "magpie", "users.json")
	b, err := os.ReadFile(usersPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "correct horse battery") {
		t.Fatal("the password is in users.json")
	}
	if info, _ := os.Stat(usersPath); info.Mode().Perm() != 0o600 {
		t.Fatalf("users.json mode %v", info.Mode())
	}
	if _, err := runWithStdin(t, "correct horse battery\n", "user", "add", "user@example.com", "--name", "Ann", "--password-stdin"); err != nil {
		t.Fatal(err)
	}
	// registration is off until an administrator turns it on, and the
	// command says which it is
	if out, err := runOut(t, "user", "registration"); err != nil || !strings.Contains(out, "closed") {
		t.Fatalf("registration status: %q %v", out, err)
	}
	if out, err := runOut(t, "user", "registration", "on"); err != nil || !strings.Contains(out, "open") {
		t.Fatalf("open registration: %q %v", out, err)
	}
	if open, err := identity.RegistrationOpen(); err != nil || !open {
		t.Fatalf("registration after on: %v %v", open, err)
	}
	if out, err := runOut(t, "user", "registration", "off"); err != nil || !strings.Contains(out, "closed") {
		t.Fatalf("close registration: %q %v", out, err)
	}
	// the same email again, case apart
	if _, err := runWithStdin(t, "correct horse battery\n", "user", "add", "USER@example.com", "--password-stdin"); !errors.Is(err, identity.ErrEmailTaken) {
		t.Fatalf("duplicate add: %v", err)
	}
	out, err = runOut(t, "user", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var list []identity.Account
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 2 {
		t.Fatalf("list %q: %v", out, err)
	}
	if list[0].Role != identity.RoleAdmin || list[1].Role != identity.RoleUser || list[1].Name != "Ann" {
		t.Fatalf("accounts %+v", list)
	}
	out, err = runOut(t, "user", "list")
	if err != nil || !strings.Contains(out, "admin@example.com") || !strings.Contains(out, "user@example.com") {
		t.Fatalf("table %q %v", out, err)
	}

	// a promotion, then the last admin kept
	if out, err := runOut(t, "user", "role", "user@example.com", "admin"); err != nil || !strings.Contains(out, "admin") {
		t.Fatalf("promote: %q %v", out, err)
	}
	if _, err := runOut(t, "user", "role", "admin@example.com", "user"); err != nil {
		t.Fatalf("demote with another admin: %v", err)
	}
	if _, err := runOut(t, "user", "role", "user@example.com", "user"); err == nil || !strings.Contains(err.Error(), "last admin") {
		t.Fatalf("demote the last admin: %v", err)
	}

	// a password reset ends the old password and the sessions
	old, _, err := identity.NewSession(userID(t, "user@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if out, err := runWithStdin(t, "another password\n", "user", "passwd", "user@example.com", "--password-stdin"); err != nil || !strings.Contains(out, "user@example.com") {
		t.Fatalf("passwd: %q %v", out, err)
	}
	if _, err := identity.Authenticate("user@example.com", "correct horse battery"); !errors.Is(err, identity.ErrBadLogin) {
		t.Fatal("the old password still signs in")
	}
	if _, err := identity.Authenticate("user@example.com", "another password"); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.SessionUser(old); !errors.Is(err, identity.ErrNoSession) {
		t.Fatal("the old session still signs in")
	}

	// disable and enable (the admin@example.com account was demoted above,
	// so it is not the last admin)
	if _, err := runOut(t, "user", "disable", "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Authenticate("admin@example.com", "correct horse battery"); !errors.Is(err, identity.ErrDisabled) {
		t.Fatalf("disabled login: %v", err)
	}
	if _, err := runOut(t, "user", "enable", "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Authenticate("admin@example.com", "correct horse battery"); err != nil {
		t.Fatal(err)
	}

	// removal needs --yes; the last admin is never removed
	if _, err := runOut(t, "user", "remove", "admin@example.com"); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("remove without --yes: %v", err)
	}
	if _, err := runOut(t, "user", "remove", "admin@example.com", "--yes"); err != nil {
		t.Fatalf("remove the demoted account: %v", err)
	}
	if _, err := runOut(t, "user", "remove", "user@example.com", "--yes"); err == nil || !strings.Contains(err.Error(), "last admin") {
		t.Fatalf("remove the last admin: %v", err)
	}
	if _, err := identity.ByEmail("admin@example.com"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("the removed account is still here")
	}
}

func userID(t *testing.T, email string) string {
	t.Helper()
	u, err := identity.ByEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	return u.ID
}
