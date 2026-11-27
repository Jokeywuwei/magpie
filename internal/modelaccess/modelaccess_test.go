package modelaccess

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func policyHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}

func relay(t *testing.T, models ...string) {
	t.Helper()
	p := provider.Provider{ID: "relay", Name: "Relay", Key: "test-upstream-key", Chat: "https://relay.example/v1", Models: models}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
}

func TestModelAccessIsClosedUntilOpened(t *testing.T) {
	policyHome(t)
	relay(t, "m")
	models := AllModels()
	if len(models) != 1 || models[0].ID != "relay/m" || models[0].Open || !models[0].Available {
		t.Fatalf("the default catalog: %+v", models)
	}
	if Allows("relay/m") || AllowsAny("relay/m") || len(OpenIDs()) != 0 || len(OpenModels()) != 0 {
		t.Fatal("a model with no rule is not closed")
	}
	if !slices.Equal(models[0].Protocols, []string{"chat"}) {
		t.Fatalf("protocols: %v", models[0].Protocols)
	}

	opened, err := Set("relay/m", true, "Public M", "A model for everyone")
	if err != nil {
		t.Fatal(err)
	}
	if !opened.Open || opened.Name != "Public M" || opened.Description != "A model for everyone" || opened.UpdatedAt.IsZero() {
		t.Fatalf("opened: %+v", opened)
	}
	if !Allows("relay/m") || !AllowsModel(provider.Provider{ID: "relay"}, "m") {
		t.Fatal("an opened model is still refused")
	}
	if got := OpenIDs(); !slices.Equal(got, []string{"relay/m"}) {
		t.Fatalf("open ids: %v", got)
	}
	if got := OpenModels(); len(got) != 1 || got[0].Name != "Public M" {
		t.Fatalf("open models: %+v", got)
	}
	// what the store keeps is the rule, private to its owner
	b, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	var f policyFile
	if err := json.Unmarshal(b, &f); err != nil || !f.Models["relay/m"].Open || f.Models["relay/m"].Name != "Public M" {
		t.Fatalf("stored %s %v", b, err)
	}
	if info, err := os.Stat(Path()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v %v", info.Mode(), err)
	}
}

func TestClosingIsImmediateAndKeepsTheAdminName(t *testing.T) {
	policyHome(t)
	relay(t, "m")
	if _, err := Set("relay/m", true, "Public M", "Written once"); err != nil {
		t.Fatal(err)
	}
	closed, err := Update("relay/m", ptr(false), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Open || closed.Name != "Public M" || closed.Description != "Written once" {
		t.Fatalf("closing rewrote the display: %+v", closed)
	}
	if Allows("relay/m") {
		t.Fatal("a closed model is open")
	}
	// clearing the admin's name falls back to the provider's, not to "".
	cleared, err := Update("relay/m", nil, ptr(""), ptr(""))
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Name == "" || cleared.Name == "Public M" || cleared.Description != "" {
		t.Fatalf("cleared display: %+v", cleared)
	}
}

func TestUnservedModelsLeaveTheDirectory(t *testing.T) {
	policyHome(t)
	relay(t, "m")
	if _, err := Set("relay/missing", true, "", ""); err == nil {
		t.Fatal("opened a model the Gateway doesn't serve")
	}
	if _, err := Set("relay/m", true, "M", ""); err != nil {
		t.Fatal(err)
	}
	if err := provider.Delete("relay"); err != nil {
		t.Fatal(err)
	}
	// its open rule is still a rule, but a model the Gateway no longer
	// serves is never in the directory a key is granted from
	for _, m := range AllModels() {
		if m.ID == "relay/m" {
			t.Fatal("a removed provider's model is listed")
		}
	}
	if slices.Contains(OpenIDs(), "relay/m") {
		t.Fatal("a removed provider's model is a key's grant")
	}
	if _, err := Set("relay/m", false, "", ""); err == nil {
		t.Fatal("changed the rule of a model the Gateway doesn't serve")
	}
}

func TestMappedModelIDsShareOneRule(t *testing.T) {
	policyHome(t)
	relay(t, "vendor-m")
	replaced, changed, err := provider.SetModelID("relay/vendor-m", "public-m")
	if err != nil || !changed || replaced != "" {
		t.Fatalf("SetModelID: %q %v %v", replaced, changed, err)
	}
	models := AllModels()
	if len(models) != 1 || models[0].ID != "relay/public-m" {
		t.Fatalf("the mapped catalog: %+v", models)
	}
	if _, err := Set("relay/public-m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	p, err := provider.Find("relay")
	if err != nil {
		t.Fatal(err)
	}
	// the rule named by the id users see accepts the vendor's own id too,
	// which a request or an older key may still use
	if !AllowsModel(*p, "public-m") || !AllowsModel(*p, "vendor-m") {
		t.Fatal("a mapped model is refused by one of its ids")
	}
	if got := OpenIDs(); !slices.Equal(got, []string{"relay/public-m"}) {
		t.Fatalf("open ids: %v", got)
	}
}

func TestRoutingGroupIsAnOpenableModel(t *testing.T) {
	policyHome(t)
	relay(t, "m")
	if err := provider.SaveGroup(provider.Group{ID: "flash", Name: "Flash", Members: []string{"relay/m"}}); err != nil {
		t.Fatal(err)
	}
	models := AllModels()
	var group *Model
	for i := range models {
		if models[i].ID == "group/flash" {
			group = &models[i]
			break
		}
	}
	if group == nil {
		t.Fatalf("the routing group is not in the directory: %+v", models)
	}
	if group.Kind != "group" || group.Name != "Flash" || group.Provider != "" || group.ProviderID != "" {
		t.Fatalf("group fields: %+v", group)
	}
	if !slices.Equal(group.Protocols, []string{"chat"}) || !group.Available || group.Open {
		t.Fatalf("group facts: %+v", group)
	}

	opened, err := Set("group/flash", true, "Public Flash", "A group for everyone")
	if err != nil {
		t.Fatal(err)
	}
	if !opened.Open || opened.Name != "Public Flash" || opened.Kind != "group" {
		t.Fatalf("opened group: %+v", opened)
	}
	if !Allows("group/flash") || !slices.Contains(OpenIDs(), "group/flash") {
		t.Fatal("an open routing group is not offered")
	}
	if Allows("relay/m") {
		t.Fatal("opening the group opened its member too")
	}
	if _, err := Set("relay/m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	if !AllowsModel(provider.Provider{ID: "relay"}, "m") {
		t.Fatal("an opened provider model is refused")
	}

	if err := SetGroupsOnly(true); err != nil {
		t.Fatal(err)
	}
	if !GroupsOnly() {
		t.Fatal("groups-only did not stay")
	}
	if got := OpenIDs(); !slices.Equal(got, []string{"group/flash"}) {
		t.Fatalf("groups-only open ids: %v", got)
	}
	if got := OpenModels(); len(got) != 1 || got[0].ID != "group/flash" {
		t.Fatalf("groups-only plaza: %+v", got)
	}
	if AllowsModel(provider.Provider{ID: "relay"}, "m") {
		t.Fatal("groups-only still let a provider model through")
	}
	var f policyFile
	b, err := os.ReadFile(Path())
	if err != nil || json.Unmarshal(b, &f) != nil || !f.GroupsOnly || !f.Models["group/flash"].Open {
		t.Fatalf("stored policy %s %v", b, err)
	}

	if err := SetGroupsOnly(false); err != nil {
		t.Fatal(err)
	}
	if got := OpenIDs(); !slices.Equal(got, []string{"group/flash", "relay/m"}) {
		t.Fatalf("both kinds of open model: %v", got)
	}
	if !AllowsModel(provider.Provider{ID: "relay"}, "m") {
		t.Fatal("turning groups-only off kept the provider model closed")
	}
}

func ptr[T any](v T) *T { return &v }
