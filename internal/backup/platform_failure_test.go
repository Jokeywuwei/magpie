package backup

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
)

func platformBytes(t *testing.T) (users, sessions, policy, registration []byte) {
	t.Helper()
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	return read(identity.Path()), read(identity.SessionsPath()), read(modelaccess.Path()), read(identity.RegistrationPath())
}

func platformFixture(t *testing.T) {
	t.Helper()
	home(t)
	admin, err := identity.Add("admin@rollback.test", "Admin", platformPassword(t), identity.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := identity.NewSession(admin.ID); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelaccess.Path(), []byte(`{"models":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := identity.SetRegistrationOpen(true); err != nil {
		t.Fatal(err)
	}
}

// A damaged platform block is refused before any file is written.
func TestPlatformRestoreRejectsBrokenJSON(t *testing.T) {
	platformFixture(t)
	users, sessions, policy, registration := platformBytes(t)
	if _, err := Restore(Bundle{Version: BundleVersion, Platform: &Platform{
		Users: []byte(`{"not":"an array"}`), Sessions: []byte(`{}`),
		ModelAccess: []byte(`{"models":{}}`), Registration: []byte(`{"open":false}`),
	}}, Parts{Platform: true}); err == nil {
		t.Fatal("a broken accounts file was restored")
	}
	for name, got := range map[string][]byte{
		"users": users, "sessions": sessions, "policy": policy, "registration": registration,
	} {
		path := map[string]string{"users": identity.Path(), "sessions": identity.SessionsPath(), "policy": modelaccess.Path(), "registration": identity.RegistrationPath()}[name]
		if b, err := os.ReadFile(path); err != nil || !bytes.Equal(b, got) {
			t.Errorf("%s changed: %v", name, err)
		}
	}
}

func TestPlatformRestoreRefusesANewerBundle(t *testing.T) {
	platformFixture(t)
	users, _, _, _ := platformBytes(t)
	_, err := Restore(Bundle{Version: BundleVersion + 1, Platform: &Platform{Users: []byte(`[]`)}}, Parts{Platform: true})
	if err == nil {
		t.Fatal("a newer bundle was restored")
	}
	if b, err := os.ReadFile(identity.Path()); err != nil || !bytes.Equal(b, users) {
		t.Fatal("the newer bundle changed accounts")
	}
}

// A write failure in the middle puts the earlier files back, so a failed
// restore leaves no half-state.
func TestPlatformRestoreRollsBackOnPartialWrite(t *testing.T) {
	platformFixture(t)
	users, sessions, policy, registration := platformBytes(t)
	old := writePlatform
	t.Cleanup(func() { writePlatform = old })
	n := 0
	writePlatform = func(path string, data []byte) error {
		n++
		if n == 3 {
			return errors.New("disk full")
		}
		return writePlatformFile(path, data)
	}
	_, err := Restore(Bundle{Version: BundleVersion, Platform: &Platform{
		Users: []byte(`[]`), Sessions: []byte(`{}`),
		ModelAccess: []byte(`{"models":{}}`), Registration: []byte(`{"open":false}`),
	}}, Parts{Platform: true})
	if err == nil {
		t.Fatal("a failed restore reported success")
	}
	for name, got := range map[string][]byte{
		"users": users, "sessions": sessions, "policy": policy, "registration": registration,
	} {
		path := map[string]string{"users": identity.Path(), "sessions": identity.SessionsPath(), "policy": modelaccess.Path(), "registration": identity.RegistrationPath()}[name]
		if b, err := os.ReadFile(path); err != nil || !bytes.Equal(b, got) {
			t.Errorf("%s was left changed: %v", name, err)
		}
	}
	// the lock/failure did not leave the test hook behind
	if n < 3 {
		t.Fatalf("the injected failure never ran (%d writes)", n)
	}
}

func TestPlatformIsLeftOutOfAKeylessBackup(t *testing.T) {
	platformFixture(t)
	b, err := Collect(false, "keyless")
	if err != nil {
		t.Fatal(err)
	}
	if b.Platform != nil {
		t.Fatal("a --no-keys backup carried platform accounts")
	}
}

// A corrupt local platform file stops the backup instead of becoming "no
// accounts" in it.
func TestCorruptPlatformFileStopsTheBackup(t *testing.T) {
	platformFixture(t)
	if err := os.WriteFile(identity.Path(), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Collect(true, "corrupt"); err == nil {
		t.Fatal("a backup was made from a corrupt users file")
	}
}

// An older bundle, or one asked not to carry platform state, leaves it
// alone.
func TestPlatformRestoreSkipsAnOlderBundle(t *testing.T) {
	platformFixture(t)
	users, _, _, _ := platformBytes(t)
	if r, err := Restore(Bundle{Version: 1}, Parts{Platform: true}); err != nil || r.Platform {
		t.Fatalf("an older bundle: %+v %v", r, err)
	}
	if b, err := os.ReadFile(identity.Path()); err != nil || !bytes.Equal(b, users) {
		t.Fatal("an older bundle changed accounts")
	}
}
