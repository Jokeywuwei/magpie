package gui

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/usage"
)

func TestUserUsageIsTheOwnersOnly(t *testing.T) {
	sandboxHome(t)
	admin, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	a, err := identity.Register("a@example.com", "A", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	b, err := identity.Register("b@example.com", "B", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.Local)
	old := usage.Clock
	usage.Clock = func() time.Time { return now }
	t.Cleanup(func() { usage.Clock = old })
	at := now.Add(-2 * time.Hour)
	usage.Append(usage.Record{Time: at, Provider: "p", Model: "m1", Status: 200, Input: 100, Output: 10, CallerKeyID: "ka", CallerKeyName: "A's laptop", CallerKeyOwner: a.ID})
	usage.Append(usage.Record{Time: at.Add(time.Minute), Provider: "p", Model: "m2", Status: 500, Error: "upstream refused", Input: 5, CallerKeyOwner: a.ID, CallerKeyName: "A's laptop"})
	usage.Append(usage.Record{Time: at.Add(2 * time.Minute), Provider: "p", Model: "m1", Status: 200, Input: 900, Output: 90, CallerKeyOwner: b.ID, CallerKeyName: "B's laptop"})
	usage.Append(usage.Record{Time: at.Add(3 * time.Minute), Provider: "p", Model: "old", Status: 200, Input: 1000, Output: 100, CallerKeyName: "Old"})
	_ = admin

	h := Handler(nil, nil)
	aCookie := authSignIn(t, h, "a@example.com", "correct horse battery")
	bCookie := authSignIn(t, h, "b@example.com", "correct horse battery")

	// no session, no usage
	if rec := authRequest(h, "GET", "/api/user/usage/summary?period=all", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("summary without a session: %d %s", rec.Code, rec.Body)
	}
	// a query that names another account doesn't change whose summary it is
	rec := authRequest(h, "GET", "/api/user/usage/summary?period=all&owner="+b.ID+"&user="+b.ID, "", aCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary: %d %s", rec.Code, rec.Body)
	}
	var sum struct {
		Period  string            `json:"period"`
		Summary usage.UserSummary `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Period != "all" || sum.Summary.Calls != 2 || sum.Summary.Success != 1 || sum.Summary.Failed != 1 || sum.Summary.Input != 105 || sum.Summary.Output != 10 || sum.Summary.Total != 115 {
		t.Fatalf("A's summary %+v", sum)
	}
	// B's is B's, and neither has the ownerless old record
	rec = authRequest(h, "GET", "/api/user/usage/summary?period=all", "", bCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("B's summary: %d %s", rec.Code, rec.Body)
	}
	var bsum struct {
		Summary usage.UserSummary `json:"summary"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &bsum)
	if bsum.Summary.Calls != 1 || bsum.Summary.Input != 900 {
		t.Fatalf("B's summary %+v", bsum)
	}
	// trend and models are the same ledger, owner-filtered
	rec = authRequest(h, "GET", "/api/user/usage/trend?period=all", "", aCookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "2026-11-22") {
		t.Fatalf("A's trend: %d %s", rec.Code, rec.Body)
	}
	rec = authRequest(h, "GET", "/api/user/usage/models?period=all", "", aCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("A's models: %d %s", rec.Code, rec.Body)
	}
	var models struct {
		Models []usage.UserModel `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &models); err != nil || len(models.Models) != 2 || models.Models[0].Model != "m1" {
		t.Fatalf("A's models %+v %v", models, err)
	}
	// the log, its pages and filters
	page := func(cookie *http.Cookie, q string) usage.UserRequests {
		rec := authRequest(h, "GET", "/api/user/usage/requests?period=all&"+q, "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("requests %s: %d %s", q, rec.Code, rec.Body)
		}
		var out usage.UserRequests
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if p := page(aCookie, "page=1&page_size=1"); p.Total != 2 || p.Pages != 2 || len(p.Requests) != 1 || p.Requests[0].Model != "m2" {
		t.Fatalf("A's first page %+v", p)
	}
	if p := page(aCookie, "page=2&page_size=1"); p.Total != 2 || len(p.Requests) != 1 || p.Requests[0].Model != "m1" {
		t.Fatalf("A's second page %+v", p)
	}
	if p := page(aCookie, "model=m2"); p.Total != 1 || p.Requests[0].State != "failed" || p.Requests[0].KeyName != "A's laptop" {
		t.Fatalf("A's model filter %+v", p)
	}
	if p := page(aCookie, "state=success"); p.Total != 1 || p.Requests[0].Model != "m1" {
		t.Fatalf("A's state filter %+v", p)
	}
	if p := page(aCookie, "owner="+b.ID+"&user="+b.ID); p.Total != 2 || p.Requests[0].KeyName != "A's laptop" {
		t.Fatalf("A's log after naming B %+v", p)
	}
	if p := page(bCookie, ""); p.Total != 1 || p.Requests[0].KeyName != "B's laptop" {
		t.Fatalf("B's log %+v", p)
	}
	// a bad period or state is refused, not guessed
	if rec := authRequest(h, "GET", "/api/user/usage/summary?period=nonsense", "", aCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("nonsense period: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/user/usage/requests?period=all&state=nonsense", "", aCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("nonsense state: %d %s", rec.Code, rec.Body)
	}

	// the administrator's platform summary still sees every record
	rec = authRequest(h, "GET", "/api/usage?period=all", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("the platform summary: %d %s", rec.Code, rec.Body)
	}
	var platform struct {
		Calls int `json:"calls"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &platform); err != nil || platform.Calls < 4 {
		t.Fatalf("the platform summary %+v %v", platform, err)
	}
}
