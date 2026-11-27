package access

import (
	"strings"
	"testing"
)

// A platform user's key must name at least one open model. An explicit
// empty list is not "every model": that is how the old gateway read an
// ownerless key, and reading it that way for a user's key handed the key
// every tenant's data.
func TestCreateUserKeyRejectsEmptyGrant(t *testing.T) {
	userKeyHome(t)
	for _, grant := range [][]string{{""}, {" "}, {}, {"", " "}, {"", "\t"}} {
		if _, _, err := CreateUserKey("owner-a", "Laptop", grant); err == nil {
			t.Errorf("an explicit empty grant %q was accepted", grant)
		}
	}
	// nil still asks for every open model: with none open it makes a key
	// that may use none, which is safe because the owner is held.
	if _, _, err := CreateUserKey("owner-a", "Laptop", nil); err != nil {
		t.Errorf("nil (every open model) was refused: %v", err)
	}
}

func TestUpdateUserKeyRejectsEmptyGrant(t *testing.T) {
	userKeyHome(t)
	k, _, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	empty := []string{""}
	if _, err := UpdateUserKey("owner-a", k.ID, nil, &empty); err == nil {
		t.Error("PATCH to an empty grant was accepted")
	}
	if _, err := SetUserKeyModels("owner-a", k.ID, []string{" "}); err == nil {
		t.Error("SetUserKeyModels to an empty grant was accepted")
	}
}

// The admin page's models-key says "no models = every model" for the keys
// magpie makes itself. For a platform user's key the same empty list means
// none, so it is refused rather than silently creating a full-scope key.
func TestAdminModelsKeyEmptyRefusedForUserKey(t *testing.T) {
	userKeyHome(t)
	k, _, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Update("models-key", Change{Key: k.ID, Models: []string{}}); err == nil {
		t.Error("models-key emptied a user's grant")
	}
	admin, err := Update("add-key", Change{Name: "Admin key"})
	if err != nil {
		t.Fatal(err)
	}
	who, ok := Authenticate(admin)
	if !ok {
		t.Fatal("the admin key doesn't authenticate")
	}
	if _, err := Update("models-key", Change{Key: who.KeyID, Models: []string{}}); err != nil {
		t.Fatalf("an ownerless key lost its same-name grant: %v", err)
	}
}

// A platform user's key can't be given MCP servers or tools this release:
// the stored list is ignored for it and the management path says so.
func TestAdminMCPKeyRefusedForUserKey(t *testing.T) {
	userKeyHome(t)
	k, secret, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Update("mcp-key", Change{Key: k.ID, MCP: []string{"magpie-tools"}}); err == nil || !strings.Contains(err.Error(), "platform user") {
		t.Fatalf("a user key was given tools: %v", err)
	}
	// the key still authenticates, with no tools
	who, ok := Authenticate(secret)
	if !ok {
		t.Fatal("the key stopped authenticating")
	}
	if who.MCPAllows("magpie-tools", "anything") {
		t.Fatal("a user key allows a tool")
	}
	admin, err := Update("add-key", Change{Name: "Admin key"})
	if err != nil {
		t.Fatal(err)
	}
	who, _ = Authenticate(admin)
	if _, err := Update("mcp-key", Change{Key: who.KeyID, MCP: []string{"magpie-tools"}}); err != nil {
		t.Fatalf("an ownerless key lost its tools: %v", err)
	}
}
