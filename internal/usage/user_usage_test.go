package usage

import (
	"strings"
	"testing"
	"time"
)

func TestUserReportIsTheOwnersOnly(t *testing.T) {
	resetUsageTest(t)
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.Local)
	at := now.Add(-2 * time.Hour)
	Append(Record{Time: at, Provider: "p", Model: "m1", Status: 200, Input: 100, Output: 10, CallerKeyID: "ka", CallerKeyName: "A's laptop", CallerKeyOwner: "a"})
	Append(Record{Time: at.Add(time.Minute), Provider: "p", Model: "m2", Status: 500, Error: "upstream refused", Input: 5, Output: 0, CallerKeyID: "ka", CallerKeyName: "A's laptop", CallerKeyOwner: "a"})
	Append(Record{Time: at.Add(2 * time.Minute), Provider: "p", Model: "m1", Status: 200, Input: 900, Output: 90, CallerKeyID: "kb", CallerKeyName: "B's laptop", CallerKeyOwner: "b"})
	// a record from before the field, and one magpie made itself: no account's
	Append(Record{Time: at.Add(3 * time.Minute), Provider: "p", Model: "old", Status: 200, Input: 1000, Output: 100, CallerKeyID: "old", CallerKeyName: "Old"})
	Append(Record{Time: at.Add(4 * time.Minute), Provider: "p", Model: "local", Status: 200, Input: 2000, Output: 200})
	if recs := Load(time.Time{}); len(recs) != 5 {
		t.Fatalf("appended %d records", len(recs))
	}
	if recs := Load(time.Time{}); len(recs) != 5 || recs[0].CallerKeyOwner != "a" || recs[1].CallerKeyOwner != "a" {
		t.Fatalf("appended or owned records: %d %+v", len(recs), recs)
	}

	if all := UserReportOf("a", All, now, time.Local); all.Summary.Calls != 2 {
		t.Fatalf("A's all-time report %+v", all.Summary)
	}
	a := UserReportOf("a", Week, now, time.Local)
	if a.Summary.Calls != 2 || a.Summary.Success != 1 || a.Summary.Failed != 1 || a.Summary.Input != 105 || a.Summary.Output != 10 || a.Summary.Total != 115 {
		t.Fatalf("A's summary %+v", a.Summary)
	}
	if len(a.Models) != 2 || a.Models[0].Model != "m1" || a.Models[0].Input != 100 {
		t.Fatalf("A's models %+v", a.Models)
	}
	if b := UserReportOf("b", Week, now, time.Local); b.Summary.Calls != 1 || b.Summary.Input != 900 || len(b.Models) != 1 || b.Models[0].Model != "m1" {
		t.Fatalf("B's report %+v", b)
	}
	// the trend has the week's days and A's tokens only
	sum := 0
	for _, d := range a.Trend {
		sum += d.Total
	}
	if len(a.Trend) == 0 || sum != 115 {
		t.Fatalf("A's trend %+v", a.Trend)
	}

	// the log is A's alone, newest first, with pagination and filters
	req := UserRequestsOf("a", Week, "", "", 1, 1, now, time.Local)
	if req.Total != 2 || req.Pages != 2 || len(req.Requests) != 1 || req.Requests[0].Model != "m2" {
		t.Fatalf("A's first page %+v", req)
	}
	if req.Requests[0].State != "failed" || req.Requests[0].KeyName != "A's laptop" || req.Requests[0].Error != "upstream refused" {
		t.Fatalf("A's failed request %+v", req.Requests[0])
	}
	req = UserRequestsOf("a", Week, "", "", 2, 1, now, time.Local)
	if req.Total != 2 || len(req.Requests) != 1 || req.Requests[0].Model != "m1" || req.Requests[0].State != "success" {
		t.Fatalf("A's second page %+v", req)
	}
	if req := UserRequestsOf("a", Week, "m2", "", 1, 20, now, time.Local); req.Total != 1 || req.Requests[0].State != "failed" {
		t.Fatalf("A's model filter %+v", req)
	}
	if req := UserRequestsOf("a", Week, "", "success", 1, 20, now, time.Local); req.Total != 1 || req.Requests[0].Model != "m1" {
		t.Fatalf("A's state filter %+v", req)
	}
	// B's log never carries A's calls, and no blank-owner record is anyone's
	if req := UserRequestsOf("b", Week, "", "", 1, 20, now, time.Local); req.Total != 1 || req.Requests[0].KeyName != "B's laptop" {
		t.Fatalf("B's log %+v", req)
	}
	if req := UserRequestsOf("nobody", Week, "", "", 1, 20, now, time.Local); req.Total != 0 {
		t.Fatalf("a stranger's log %+v", req)
	}
}

func TestUserRequestHidesSecretsAndLimitsError(t *testing.T) {
	resetUsageTest(t)
	now := time.Now()
	Append(Record{Time: now, Provider: "p", Model: "m", Status: 502, Error: "vendor said Authorization: Bearer sk-abcdefgh12345678 and gsk_abcdefgh12345678\nsecond line", CallerKeyOwner: "a"})
	req := UserRequestsOf("a", Today, "", "", 1, 20, now, time.Local)
	if len(req.Requests) != 1 {
		t.Fatalf("%+v", req)
	}
	e := req.Requests[0].Error
	if strings.Contains(e, "abcdefgh12345678") || strings.Contains(e, "gsk_abcdefgh12345678") || strings.Contains(e, "\n") {
		t.Fatalf("the error still holds a secret or a line break: %q", e)
	}
	if !strings.Contains(e, "sk-…") || !strings.Contains(e, "gsk_…") || !strings.Contains(e, "Bearer") {
		t.Fatalf("the error masks what it should: %q", e)
	}
	if len([]rune(e)) > 241 {
		t.Fatalf("the error wasn't cut: %d runes", len([]rune(e)))
	}
}

// A successful call records the provider's short model id; a refused call
// records only the id the client asked for. The ranking and the log show
// both by the requested platform id, while the API keeps the recorded id
// and the original request id separate.
func TestUserModelDisplayUsesTheRequestedID(t *testing.T) {
	resetUsageTest(t)
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.Local)
	Append(Record{Time: now, Provider: "deepseek-test", Model: "phase7-flash", Requested: "deepseek-test/phase7-flash", Status: 200, Input: 10, Output: 2, CallerKeyOwner: "a"})
	Append(Record{Time: now.Add(time.Second), Provider: "deepseek-test", Requested: "deepseek-test/phase7-flash", Status: 403, CallerKeyOwner: "a"})
	Append(Record{Time: now.Add(2 * time.Second), Provider: "deepseek-test", Requested: "deepseek-test/phase7-pro", Status: 403, CallerKeyOwner: "a"})

	report := UserReportOf("a", Today, now, time.Local)
	if len(report.Models) != 2 {
		t.Fatalf("models %+v", report.Models)
	}
	by := map[string]UserModel{}
	for _, m := range report.Models {
		by[m.Model] = m
	}
	if m := by["deepseek-test/phase7-flash"]; m.Calls != 2 || m.Total != 12 {
		t.Fatalf("flash ranking %+v", m)
	}
	if m := by["deepseek-test/phase7-pro"]; m.Calls != 1 || m.Total != 0 {
		t.Fatalf("pro ranking %+v", m)
	}
	for _, m := range report.Models {
		if strings.HasPrefix(m.Model, "phase7-") {
			t.Fatalf("a short id reached the ranking: %+v", m)
		}
	}

	req := UserRequestsOf("a", Today, "deepseek-test/phase7-flash", "", 1, 20, now, time.Local)
	if req.Total != 2 {
		t.Fatalf("canonical filter %+v", req)
	}
	// the API keeps both the recorded model and the requested id
	success := req.Requests[1]
	if success.State != "success" || success.Model != "phase7-flash" || success.DisplayModel != "deepseek-test/phase7-flash" || success.Requested != "deepseek-test/phase7-flash" {
		t.Fatalf("successful request %+v", success)
	}
	refused := req.Requests[0]
	if refused.State != "failed" || refused.DisplayModel != "deepseek-test/phase7-flash" || refused.Requested != "deepseek-test/phase7-flash" {
		t.Fatalf("refused request %+v", refused)
	}
	// the old filter by the recorded short id still selects the call it named
	if old := UserRequestsOf("a", Today, "phase7-flash", "", 1, 20, now, time.Local); old.Total != 1 || old.Requests[0].State != "success" {
		t.Fatalf("recorded-id filter %+v", old)
	}
	if pro := UserRequestsOf("a", Today, "deepseek-test/phase7-pro", "", 1, 20, now, time.Local); pro.Total != 1 || pro.Requests[0].State != "failed" {
		t.Fatalf("pro filter %+v", pro)
	}
}
