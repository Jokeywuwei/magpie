package access

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func userKeyHome(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestUserKeyIsHashedAndOwned(t *testing.T) {
	userKeyHome(t)
	k, secret, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, UserPrefix) || !UserSecret(secret) || k.ID == "" || k.Prefix == "" || k.Off {
		t.Fatalf("made %+v %q", k, secret)
	}
	// the store keeps its hash, not the secret
	b, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) {
		t.Fatal("the secret is in caller-keys.json")
	}
	var keys []Key
	if err := json.Unmarshal(b, &keys); err != nil || len(keys) != 1 {
		t.Fatalf("%v %v", keys, err)
	}
	if keys[0].Owner != "owner-a" || len(keys[0].Hash) != 64 || keys[0].Secret != "" {
		t.Fatalf("stored %+v", keys[0])
	}
	if info, err := os.Stat(Path()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v %v", info.Mode(), err)
	}
	// and it signs in as that account, and only that one
	who, err := Check(secret)
	if err != nil || who.KeyID != k.ID || who.Owner != "owner-a" || who.Admin != "" || who.Memory || who.Issues || who.Default {
		t.Fatalf("checked %+v %v", who, err)
	}
	if who, err := Check(secret + "x"); err == nil {
		t.Fatalf("a wrong secret signed in: %+v", who)
	}
	// the list never carries the hash or the secret
	list, err := List()
	if err != nil || len(list) != 1 || list[0].Secret != "" || list[0].Hash != "" || !strings.HasPrefix(list[0].Masked, k.Prefix) {
		t.Fatalf("list %+v %v", list, err)
	}
	// last use is kept, and is not in the future
	keys, _ = load()
	if keys[0].LastUsed.IsZero() || keys[0].LastUsed.After(time.Now().Add(time.Second)) {
		t.Fatalf("last use %v", keys[0].LastUsed)
	}
}

func TestUserKeysAreIsolated(t *testing.T) {
	userKeyHome(t)
	a, aSecret, err := CreateUserKey("owner-a", "A's key", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, bSecret, err := CreateUserKey("owner-b", "B's key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if list, _ := UserKeys("owner-a"); len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("A's list %+v", list)
	}
	if list, _ := UserKeys("owner-b"); len(list) != 1 || list[0].ID != b.ID {
		t.Fatalf("B's list %+v", list)
	}
	if _, err := SetUserKeyOff("owner-b", a.ID, true); !errors.Is(err, ErrNoKey) {
		t.Fatalf("B turned off A's key: %v", err)
	}
	if err := DeleteUserKey("owner-b", a.ID); !errors.Is(err, ErrNoKey) {
		t.Fatalf("B removed A's key: %v", err)
	}
	if who, err := Check(aSecret); err != nil || who.Owner != "owner-a" {
		t.Fatalf("A's key after B's tries: %+v %v", who, err)
	}
	// off, then on, then gone
	if _, err := SetUserKeyOff("owner-a", a.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(aSecret); err == nil {
		t.Fatal("a turned-off user key still signs in")
	}
	if _, err := SetUserKeyOff("owner-a", a.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(aSecret); err != nil {
		t.Fatalf("a turned-on key doesn't sign in: %v", err)
	}
	if err := DeleteUserKey("owner-a", a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(aSecret); err == nil {
		t.Fatal("a deleted user key still signs in")
	}
	if _, err := Check(bSecret); err != nil {
		t.Fatalf("B's key after A's removal: %v", err)
	}
	if n, err := DeleteUserKeys("owner-b"); err != nil || n != 1 {
		t.Fatalf("DeleteUserKeys: %d %v", n, err)
	}
	if _, err := Check(bSecret); err == nil {
		t.Fatal("B's key after its account's keys went")
	}
}

func TestUserKeyCannotBeMadeAnAdmin(t *testing.T) {
	userKeyHome(t)
	k, secret, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	// even if a hand-written or older management call sets them, the
	// identity a user's key gives is never an admin's
	if _, err := Update("admin-key", Change{Key: k.ID, Admin: AdminAny}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update("memory-key", Change{Key: k.ID, Memory: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update("issuer-key", Change{Key: k.ID, Issues: true}); err != nil {
		t.Fatal(err)
	}
	who, err := Check(secret)
	if err != nil {
		t.Fatal(err)
	}
	if who.Admin != "" || who.Memory || who.Issues || who.Browser || who.ReadOnly {
		t.Fatalf("a user's key holds management: %+v", who)
	}
}

func TestUserKeyRotationStaysHashed(t *testing.T) {
	userKeyHome(t)
	k, old, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := Update("rotate-key", Change{Key: k.ID})
	if err != nil {
		t.Fatal(err)
	}
	if fresh == old || fresh == "" {
		t.Fatalf("rotated to %q", fresh)
	}
	if _, err := Check(old); err == nil {
		t.Fatal("the old secret still signs in")
	}
	who, err := Check(fresh)
	if err != nil || who.Owner != "owner-a" {
		t.Fatalf("the new secret: %+v %v", who, err)
	}
	b, _ := os.ReadFile(Path())
	if strings.Contains(string(b), fresh) {
		t.Fatal("a rotated user key is plaintext in the store")
	}
}

func TestUserKeyLimit(t *testing.T) {
	userKeyHome(t)
	for i := 0; i < MaxUserKeys; i++ {
		if _, _, err := CreateUserKey("owner-a", "k", nil); err != nil {
			t.Fatalf("key %d: %v", i, err)
		}
	}
	if _, _, err := CreateUserKey("owner-a", "one more", nil); err == nil {
		t.Fatal("made more keys than the limit")
	}
	if _, _, err := CreateUserKey("owner-b", "B's first", nil); err != nil {
		t.Fatalf("another account at the limit: %v", err)
	}
}

func TestLegacyKeyStillWorks(t *testing.T) {
	userKeyHome(t)
	secret, err := Update("add-key", Change{Name: "Old"})
	if err != nil {
		t.Fatal(err)
	}
	who, err := Check(secret)
	if err != nil || who.Owner != "" || who.KeyName != "Old" {
		t.Fatalf("legacy key: %+v %v", who, err)
	}
	if _, err := Update("admin-key", Change{Key: who.KeyID, Admin: AdminAny}); err != nil {
		t.Fatal(err)
	}
	who, err = Check(secret)
	if err != nil || who.Admin != AdminAny {
		t.Fatalf("legacy admin key: %+v %v", who, err)
	}
}

func TestExistingAdminActionsManageAUserKey(t *testing.T) {
	userKeyHome(t)
	k, secret, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	list, err := List()
	if err != nil || len(list) != 1 || list[0].Owner != "owner-a" || list[0].Masked == "" {
		t.Fatalf("the admin's list %+v %v", list, err)
	}
	if _, err := Update("off-key", Change{Key: k.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(secret); err == nil {
		t.Fatal("the admin's off-key left a user key working")
	}
	if _, err := Update("on-key", Change{Key: k.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(secret); err != nil {
		t.Fatalf("the admin's on-key: %v", err)
	}
	if _, err := Update("remove-key", Change{Key: k.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(secret); err == nil {
		t.Fatal("the admin's remove-key left a user key working")
	}
}
