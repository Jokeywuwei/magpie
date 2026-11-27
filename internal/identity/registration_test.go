package identity

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRegistrationIsOffUntilTurnedOn(t *testing.T) {
	testHome(t)
	if open, err := RegistrationOpen(); err != nil || open {
		t.Fatalf("registration without a file: %v %v", open, err)
	}
	if err := SetRegistrationOpen(true); err != nil {
		t.Fatal(err)
	}
	if open, err := RegistrationOpen(); err != nil || !open {
		t.Fatalf("registration after opening: %v %v", open, err)
	}
	if info, err := os.Stat(RegistrationPath()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("registration.json mode %v %v", info.Mode(), err)
	}
	if err := SetRegistrationOpen(false); err != nil {
		t.Fatal(err)
	}
	if open, _ := RegistrationOpen(); open {
		t.Fatal("registration stayed open")
	}
	// a file that can't be read is closed, never open
	if err := os.WriteFile(RegistrationPath(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if open, err := RegistrationOpen(); err == nil || open {
		t.Fatalf("broken registration file: %v %v", open, err)
	}
}

func TestRoleChangeEndsTheSessionsOfTheOldRole(t *testing.T) {
	testHome(t)
	admin := mustAdmin(t, "admin@example.com")
	if _, err := Add("second@example.com", "", "correct horse battery", RoleAdmin); err != nil {
		t.Fatal(err)
	}
	user, err := Register("user@example.com", "", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	adminToken, _, err := NewSession(admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	userToken, _, err := NewSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	// a promotion doesn't turn the user's old browser into an admin
	if _, err := SetRole(user.ID, RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := SessionUser(userToken); !errors.Is(err, ErrNoSession) {
		t.Fatalf("the old user session after promotion: %v", err)
	}
	if _, err := SessionUser(adminToken); err != nil {
		t.Fatalf("another admin's session: %v", err)
	}
	// a demotion ends the admin's old browser too
	if _, err := SetRole(admin.ID, RoleUser); err != nil {
		t.Fatal(err)
	}
	if _, err := SessionUser(adminToken); !errors.Is(err, ErrNoSession) {
		t.Fatalf("the old admin session after demotion: %v", err)
	}
	// and a session made after the change carries the new role
	admin2, err := ByID(admin.ID)
	if err != nil || admin2.Role != RoleUser {
		t.Fatalf("after demotion %+v %v", admin2, err)
	}
}

func TestRegistrationFileIsNotAUserFile(t *testing.T) {
	testHome(t)
	if err := SetRegistrationOpen(true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(RegistrationPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "password") || strings.Contains(string(b), "session") {
		t.Fatalf("registration.json holds more than the switch: %s", b)
	}
}
