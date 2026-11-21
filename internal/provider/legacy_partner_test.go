package provider

import "testing"

// A provider added from a partner's preset before the add sheet's Partners
// were removed keeps working: its preset is no longer one of magpie's, but
// normalize leaves the endpoints, catalog and key it was saved with, and
// nothing drops it from providers.json for an id no preset knows.
func TestLegacyPartnerProviderKept(t *testing.T) {
	if pr := Preset("acme-partner"); pr != nil {
		t.Fatalf("acme-partner is a preset again: %+v", pr)
	}
	p := Provider{
		ID: "acme-partner", Name: "Acme", Preset: "acme-partner",
		Chat: "https://api.acme.example/v1", Anthropic: "https://api.acme.example",
		Catalog: "acme", Icon: "acme", Key: "sk-acme",
	}
	got := normalize(p)
	if got.Preset != p.Preset || got.Chat != p.Chat || got.Anthropic != p.Anthropic || got.Catalog != p.Catalog || got.Icon != p.Icon || got.Key != p.Key {
		t.Fatalf("normalize changed a legacy partner provider: %+v", got)
	}
}
