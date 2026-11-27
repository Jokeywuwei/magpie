package backup

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
	"github.com/yetone/magpie/internal/provider"
)

func platformPassword(t *testing.T) string {
	t.Helper()
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// A backup carries the platform's own state: accounts, sessions, the open
// model policy and the registration switch. Restoring on a clean machine
// must leave the same administrator, user, session, key ownership and
// model access. Before this, a restore brought providers and gateway keys
// but no accounts or policy, so every user key's owner was gone.
func TestPlatformStateTravelsWithABackup(t *testing.T) {
	home(t)
	adminPW := platformPassword(t)
	userPW := platformPassword(t)
	if _, err := identity.Add("admin@platform.test", "Admin", adminPW, identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	user, err := identity.Register("user@platform.test", "User", userPW)
	if err != nil {
		t.Fatal(err)
	}
	if err := identity.SetRegistrationOpen(true); err != nil {
		t.Fatal(err)
	}
	token, _, err := identity.NewSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`)
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "upstream", Chat: up.URL + "/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := modelaccess.Set("relay/m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	_, secret, err := access.CreateUserKey(user.ID, "Laptop", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}

	b, err := Collect(true, "platform-test")
	if err != nil {
		t.Fatal(err)
	}
	if b.Version != BundleVersion {
		t.Fatalf("a new backup says version %d, want %d", b.Version, BundleVersion)
	}
	pass := platformPassword(t)
	data, err := Seal(b, pass)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(data, pass)
	if err != nil {
		t.Fatal(err)
	}

	home(t) // a clean machine, no accounts or policy
	if _, err := Restore(opened, All); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Authenticate("admin@platform.test", adminPW); err != nil {
		t.Fatalf("the administrator did not come back: %v", err)
	}
	if _, err := identity.Authenticate("user@platform.test", userPW); err != nil {
		t.Fatalf("the user did not come back: %v", err)
	}
	if _, err := identity.SessionUser(token); err != nil {
		t.Fatalf("the session did not come back: %v", err)
	}
	who, err := access.Check(secret)
	if err != nil || who.Owner != user.ID {
		t.Fatalf("the key's owner is gone: %+v, %v", who, err)
	}
	if !modelaccess.Allows("relay/m") {
		t.Fatal("the open model policy did not come back")
	}
	if open, err := identity.RegistrationOpen(); err != nil || !open {
		t.Fatalf("the registration switch did not come back: %v, %v", open, err)
	}
	for _, path := range []string{identity.Path(), identity.SessionsPath(), modelaccess.Path(), identity.RegistrationPath()} {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if st.Mode().Perm() != 0o600 {
			t.Errorf("%s: mode %o, want 600", path, st.Mode().Perm())
		}
	}
}
