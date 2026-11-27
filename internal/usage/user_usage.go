// A platform account's own usage, read through the same ledger the
// administrator's pages read: the owner is always a server-side filter, a
// number the browser never sends. A record with no owner is no account's
// and never appears on a user's page.
package usage

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/lazyre"
)

// UserSummary is one account's counts over a period.
type UserSummary struct {
	Calls    int `json:"calls"`
	Success  int `json:"success"`
	Failed   int `json:"failed"`
	Canceled int `json:"canceled"`
	Input    int `json:"input"`
	Output   int `json:"output"`
	Total    int `json:"total"`
	// CacheRead and CacheWrite are the prompt tokens vendors counted as
	// cache, kept apart from Input/Output and from Total as the Usage page
	// keeps them.
	CacheRead  int `json:"cacheRead,omitempty"`
	CacheWrite int `json:"cacheWrite,omitempty"`
}

// UserTrendPoint is one day of an account's usage.
type UserTrendPoint struct {
	Date     string `json:"date"` // YYYY-MM-DD in the reader's zone
	Calls    int    `json:"calls"`
	Success  int    `json:"success"`
	Failed   int    `json:"failed"`
	Input    int    `json:"input"`
	Output   int    `json:"output"`
	Total    int    `json:"total"`
	Canceled int    `json:"canceled,omitempty"`
}

// UserModel is one model's use by an account.
type UserModel struct {
	Model  string `json:"model"`
	Calls  int    `json:"calls"`
	Input  int    `json:"input"`
	Output int    `json:"output"`
	Total  int    `json:"total"`
}

// UserReport is an account's summary, daily trend and model ranking for a
// period.
type UserReport struct {
	Period  string           `json:"period"`
	Summary UserSummary      `json:"summary"`
	Trend   []UserTrendPoint `json:"trend"`
	Models  []UserModel      `json:"models"`
}

// UserRequest is one request in an account's log: no key secret, no
// provider credential, and an error cut to what a reader can use.
type UserRequest struct {
	Time time.Time `json:"time"`
	// Model is the model the gateway recorded for the call, when it
	// reached one; Requested is the id the client asked for. DisplayModel
	// is the one id both a successful and a refused call are shown by:
	// the requested id when there is one, the recorded model otherwise.
	Model        string `json:"model"`
	DisplayModel string `json:"displayModel,omitempty"`
	Requested    string `json:"requested,omitempty"`
	Provider     string `json:"provider,omitempty"`
	KeyName      string `json:"keyName,omitempty"`
	Status       int    `json:"status"`
	State        string `json:"state"` // success, failed or canceled
	Input        int    `json:"input"`
	Output       int    `json:"output"`
	Total        int    `json:"total"`
	Millis       int64  `json:"millis"`
	Error        string `json:"error,omitempty"`
}

// UserRequests is one page of an account's log.
type UserRequests struct {
	Requests []UserRequest `json:"requests"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Pages    int           `json:"pages"`
}

// UserPageSize bounds one page of a user's log.
const UserPageSize = 20
const UserPageSizeMost = 100

func ownerFilter(owner string, zone *time.Location) Filter {
	return Filter{Owner: owner, Zone: zone}
}

// UserReportOf is owner's summary, trend and models for p, read at now.
// The owner is never taken from anything the caller was sent.
func UserReportOf(owner string, p Period, now time.Time, zone *time.Location) UserReport {
	if zone == nil {
		zone = time.Local
	}
	l := LedgerOfAt(p, ownerFilter(owner, zone), now)
	out := UserReport{Period: string(p), Trend: []UserTrendPoint{}, Models: []UserModel{}}
	byDay := map[string]*UserTrendPoint{}
	byModel := map[string]*UserModel{}
	for _, r := range l.Rows {
		addUserCounts(&out.Summary, r)
		day := r.Time.In(zone).Format(time.DateOnly)
		d := byDay[day]
		if d == nil {
			d = &UserTrendPoint{Date: day}
			byDay[day] = d
		}
		addTrend(d, r)
		model := displayModel(r)
		m := byModel[model]
		if m == nil {
			m = &UserModel{Model: model}
			byModel[model] = m
		}
		m.Calls++
		m.Input += r.Input
		m.Output += r.Output
		m.Total += r.Input + r.Output
	}
	out.Trend = userTrend(p, now, zone, byDay)
	for _, m := range byModel {
		out.Models = append(out.Models, *m)
	}
	slices.SortFunc(out.Models, func(a, b UserModel) int {
		if c := cmp.Compare(b.Total, a.Total); c != 0 {
			return c
		}
		if c := cmp.Compare(b.Calls, a.Calls); c != 0 {
			return c
		}
		return strings.Compare(a.Model, b.Model)
	})
	if len(out.Models) > 50 {
		out.Models = out.Models[:50]
	}
	return out
}

func addUserCounts(s *UserSummary, r Row) {
	s.Calls++
	switch {
	case r.Failed():
		s.Failed++
	case r.Canceled():
		s.Canceled++
	default:
		s.Success++
	}
	s.Input += r.Input
	s.Output += r.Output
	s.Total += r.Input + r.Output
	s.CacheRead += r.CacheRead
	s.CacheWrite += r.CacheWrite
}

func addTrend(d *UserTrendPoint, r Row) {
	d.Calls++
	switch {
	case r.Failed():
		d.Failed++
	case r.Canceled():
		d.Canceled++
	default:
		d.Success++
	}
	d.Input += r.Input
	d.Output += r.Output
	d.Total += r.Input + r.Output
}

// userTrend is byDay in order, with the empty days of a bounded period
// kept: a week's chart has seven days even when two had no calls. An
// unbounded period is only the days that have calls, so a long history
// doesn't draw thousands of empty ones.
func userTrend(p Period, now time.Time, zone *time.Location, byDay map[string]*UserTrendPoint) []UserTrendPoint {
	since, until := p.Since(now), p.Until(now)
	var start, end time.Time
	bounded := false
	if !since.IsZero() {
		start = since.In(zone)
		if until.IsZero() {
			until = now // a preset ends now, not at a picked range's midnight
		}
		end = until.In(zone)
		bounded = !end.Before(start) && end.Sub(start) <= 370*24*time.Hour
	}
	out := make([]UserTrendPoint, 0, len(byDay))
	if !bounded {
		for _, d := range byDay {
			out = append(out, *d)
		}
		slices.SortFunc(out, func(a, b UserTrendPoint) int { return strings.Compare(a.Date, b.Date) })
		return out
	}
	for d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, zone); d.Before(end); d = d.AddDate(0, 0, 1) {
		day := d.Format(time.DateOnly)
		if got := byDay[day]; got != nil {
			out = append(out, *got)
		} else {
			out = append(out, UserTrendPoint{Date: day})
		}
	}
	return out
}

// UserRequestsOf is one page of owner's requests for p, filtered by model
// and state ("" or all, success, failed, canceled). The owner is a
// server-side argument, never a query field.
func UserRequestsOf(owner string, p Period, model, state string, page, size int, now time.Time, zone *time.Location) UserRequests {
	if zone == nil {
		zone = time.Local
	}
	if size <= 0 {
		size = UserPageSize
	}
	if size > UserPageSizeMost {
		size = UserPageSizeMost
	}
	if page < 1 {
		page = 1
	}
	f := ownerFilter(owner, zone)
	l := LedgerOfAt(p, f, now)
	rows := make([]Row, 0, len(l.Rows))
	for _, r := range l.Rows {
		if model != "" && !userModelMatches(r, model) {
			continue
		}
		if state != "" && state != "all" && userRequestState(r) != state {
			continue
		}
		rows = append(rows, r)
	}
	slices.SortStableFunc(rows, func(a, b Row) int { return b.Time.Compare(a.Time) })
	total := len(rows)
	pages := (total + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * size
	end := min(start+size, total)
	out := UserRequests{Requests: []UserRequest{}, Total: total, Page: page, PageSize: size, Pages: pages}
	for _, r := range rows[start:end] {
		out.Requests = append(out.Requests, userRequest(r))
	}
	return out
}

func userRequestState(r Row) string {
	switch {
	case r.Failed():
		return "failed"
	case r.Canceled():
		return "canceled"
	default:
		return "success"
	}
}

// displayModel is the one id a user sees a call by, successful or refused:
// the id the client asked for, else the model the gateway recorded. The two
// are kept apart in the API as well, so a client's own request id and the
// recorded model are never lost.
func displayModel(r Row) string {
	return cmp.Or(r.Requested, r.Model, "unknown")
}

// userModelMatches is the log's model filter: a canonical display id, a
// recorded model or an original request id all select the call, so a filter
// built from the ranking and one kept from an older page both work.
func userModelMatches(r Row, filter string) bool {
	return filter == displayModel(r) || filter == r.Model || filter == r.Requested || sameGroupModel(r.Model, filter)
}

func userRequest(r Row) UserRequest {
	q := UserRequest{
		Time: r.Time, Model: cmp.Or(r.Model, r.Requested, "unknown"), DisplayModel: displayModel(r), Requested: r.Requested,
		Provider: r.Provider, KeyName: r.CallerKeyName, Status: r.Status, State: userRequestState(r),
		Input: r.Input, Output: r.Output, Total: r.Input + r.Output, Millis: r.Millis,
	}
	q.Error = safeUserError(r.Error)
	return q
}

// safeUserError is an error as a user's log shows it: one line, at most 240
// characters, with anything shaped like a bearer token or a key's secret
// masked. A vendor sometimes echoes what it was sent in its refusal.
var (
	keyish    = lazyre.New(`(?i)\b(sk-|gsk_|xai-)[A-Za-z0-9._~+/=-]{8,}`)
	bearerish = lazyre.New(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{8,}`)
	linespace = lazyre.New(`[\r\n\t]+`)
)

func safeUserError(s string) string {
	s = strings.TrimSpace(linespace.ReplaceAllString(s, " "))
	if s == "" {
		return ""
	}
	s = keyish.ReplaceAllStringFunc(s, func(m string) string {
		low := strings.ToLower(m)
		for _, p := range []string{"sk-", "gsk_", "xai-"} {
			if strings.HasPrefix(low, p) {
				return m[:len(p)] + "…"
			}
		}
		return "…"
	})
	s = bearerish.ReplaceAllString(s, "Bearer …")
	if r := []rune(s); len(r) > 240 {
		s = string(r[:240]) + "…"
	}
	return s
}
