package access

import (
	"errors"
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/modelaccess"
	"github.com/yetone/magpie/internal/provider"
)

func grantHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}

func grantRelay(t *testing.T, models ...string) {
	t.Helper()
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "test-upstream-key", Chat: "https://relay.example/v1", Models: models}); err != nil {
		t.Fatal(err)
	}
}

func TestUserKeyWithNoModelsMayUseNone(t *testing.T) {
	grantHome(t)
	k, secret, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Models) != 0 {
		t.Fatalf("a key with no open models: %+v", k)
	}
	who, err := Check(secret)
	if err != nil {
		t.Fatal(err)
	}
	if !who.Restricted() || who.Allows("relay/m") || who.Allows() {
		t.Fatalf("an empty user grant is every model: %+v", who)
	}
	// nil asks for every open model; with none open the key may use none.
	// An explicit empty list is refused rather than creating another one.
	if _, _, err := CreateUserKey("owner-a", "Empty", []string{}); err == nil {
		t.Fatal("an explicit empty grant was accepted")
	}
}

func TestUserKeyModelsMustBeOpen(t *testing.T) {
	grantHome(t)
	grantRelay(t, "m")
	if _, _, err := CreateUserKey("owner-a", "Laptop", []string{"relay/m"}); err == nil {
		t.Fatal("made a key for a model the platform hasn't opened")
	}
	if _, err := modelaccess.Set("relay/m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	k, secret, err := CreateUserKey("owner-a", "Laptop", []string{"relay/m", "relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(k.Models, []string{"relay/m"}) {
		t.Fatalf("the grant: %v", k.Models)
	}
	who, err := Check(secret)
	if err != nil || !who.Allows("relay/m") || who.Allows("relay/other") {
		t.Fatalf("who: %+v %v", who, err)
	}
	// a key cannot be pointed at a closed model by an update either
	if _, err := UpdateUserKey("owner-a", k.ID, nil, &[]string{"relay/closed"}); err == nil {
		t.Fatal("updated a key to a closed model")
	}
}

func TestLegacyUserKeyFreezesTheOpenModelsOnce(t *testing.T) {
	grantHome(t)
	grantRelay(t, "m", "n")
	if _, err := modelaccess.Set("relay/m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	k, secret, err := CreateUserKey("owner-a", "Laptop", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	// the store as Phase 3 left it: owned, hashed, and no model list
	keys, err := load()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(keys, func(x Key) bool { return x.ID == k.ID })
	if i < 0 {
		t.Fatal("the key is missing")
	}
	keys[i].Models, keys[i].ModelsSet, keys[i].LegacyModels = nil, false, false
	if err := save(keys); err != nil {
		t.Fatal(err)
	}

	who, err := Check(secret)
	if err != nil || !slices.Equal(who.Models, []string{"relay/m"}) {
		t.Fatalf("first use: %+v %v", who, err)
	}
	keys, _ = load()
	if k := keys[i]; !k.ModelsSet || !k.LegacyModels || !slices.Equal(k.Models, []string{"relay/m"}) {
		t.Fatalf("the frozen grant: %+v", k)
	}
	// a model opened later is not silently added to the old key
	if _, err := modelaccess.Set("relay/n", true, "", ""); err != nil {
		t.Fatal(err)
	}
	who, err = Check(secret)
	if err != nil || !slices.Equal(who.Models, []string{"relay/m"}) {
		t.Fatalf("after another model opened: %+v %v", who, err)
	}
	// the console may then grant the new one explicitly
	if _, err := UpdateUserKey("owner-a", k.ID, nil, &[]string{"relay/n"}); err != nil {
		t.Fatal(err)
	}
	who, err = Check(secret)
	if err != nil || !slices.Equal(who.Models, []string{"relay/n"}) {
		t.Fatalf("after an explicit update: %+v %v", who, err)
	}
}

func TestLegacyUserKeyWithNothingOpenGetsNoGrant(t *testing.T) {
	grantHome(t)
	grantRelay(t, "m")
	k, secret, err := CreateUserKey("owner-a", "Laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	keys, _ := load()
	i := slices.IndexFunc(keys, func(x Key) bool { return x.ID == k.ID })
	keys[i].Models, keys[i].ModelsSet, keys[i].LegacyModels = nil, false, false
	if err := save(keys); err != nil {
		t.Fatal(err)
	}
	who, err := Check(secret)
	if err != nil {
		t.Fatal(err)
	}
	if len(who.Models) != 0 || !who.Restricted() || who.Allows("relay/m") {
		t.Fatalf("a legacy key with nothing open: %+v", who)
	}
	keys, _ = load()
	if !keys[i].ModelsSet || !keys[i].LegacyModels || len(keys[i].Models) != 0 {
		t.Fatalf("stored: %+v", keys[i])
	}
}

func TestAdministratorKeysKeepEveryModel(t *testing.T) {
	grantHome(t)
	secret, err := Update("add-key", Change{Name: "Platform admin"})
	if err != nil {
		t.Fatal(err)
	}
	who, err := Check(secret)
	if err != nil {
		t.Fatal(err)
	}
	if who.Owner != "" || who.Restricted() || !who.Allows("relay/anything") {
		t.Fatalf("an ownerless key changed: %+v", who)
	}
	// and only a user's key is found by the user-key APIs
	if _, err := UpdateUserKey("owner-a", who.KeyID, nil, &[]string{}); !errors.Is(err, ErrNoKey) {
		t.Fatalf("a user route touched an ownerless key: %v", err)
	}
}
