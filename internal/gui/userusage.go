// The signed-in account's own usage: the administrator's ledger read with
// the account as a server-side filter. No request field can change whose
// records these are, and a record with no owner belongs to no account.
package gui

import (
	"net/http"
	"strconv"
	"time"

	"github.com/yetone/magpie/internal/usage"
)

func parsePositive(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}

// userUsagePeriod reads the period of a user's usage. period is today, 7d,
// 30d or all; a picked range may be given as from and to (YYYY-MM-DD)
// instead. Anything else is refused, not guessed.
func userUsagePeriod(r *http.Request) (usage.Period, bool) {
	q := r.URL.Query()
	if from, to := q.Get("from"), q.Get("to"); from != "" || to != "" {
		if from == "" || to == "" {
			return "", false
		}
		p, ok := usage.RangeOf(from, to)
		return p, ok
	}
	p := usage.Period(q.Get("period"))
	if p == "" {
		p = usage.Week
	}
	return p, p.Known()
}

func userUsageReport(r *http.Request) (usage.UserReport, bool) {
	p, ok := userUsagePeriod(r)
	if !ok {
		return usage.UserReport{}, false
	}
	u, _ := userOf(r)
	return usage.UserReportOf(u.ID, p, usage.Clock(), time.Local), true
}

func userUsageRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/user/usage/summary", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rep, ok := userUsageReport(r)
		if !ok {
			authFail(rw, http.StatusBadRequest, "period is today, 7d, 30d or all, or a from and to date")
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"period": rep.Period, "summary": rep.Summary})
	})).ServeHTTP)
	mux.HandleFunc("GET /api/user/usage/trend", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rep, ok := userUsageReport(r)
		if !ok {
			authFail(rw, http.StatusBadRequest, "period is today, 7d, 30d or all, or a from and to date")
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"period": rep.Period, "trend": rep.Trend})
	})).ServeHTTP)
	mux.HandleFunc("GET /api/user/usage/models", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rep, ok := userUsageReport(r)
		if !ok {
			authFail(rw, http.StatusBadRequest, "period is today, 7d, 30d or all, or a from and to date")
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"period": rep.Period, "models": rep.Models})
	})).ServeHTTP)
	mux.HandleFunc("GET /api/user/usage/requests", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		p, ok := userUsagePeriod(r)
		if !ok {
			authFail(rw, http.StatusBadRequest, "period is today, 7d, 30d or all, or a from and to date")
			return
		}
		q := r.URL.Query()
		state := q.Get("state")
		switch state {
		case "", "all", "success", "failed", "canceled":
		default:
			authFail(rw, http.StatusBadRequest, "state is success, failed, canceled or all")
			return
		}
		page, size := 1, usage.UserPageSize
		if v := q.Get("page"); v != "" {
			n, err := parsePositive(v)
			if err != nil {
				authFail(rw, http.StatusBadRequest, "page is a positive number")
				return
			}
			page = n
		}
		if v := q.Get("page_size"); v != "" {
			n, err := parsePositive(v)
			if err != nil {
				authFail(rw, http.StatusBadRequest, "page_size is a positive number")
				return
			}
			size = n
		}
		u, _ := userOf(r)
		out := usage.UserRequestsOf(u.ID, p, q.Get("model"), state, page, size, usage.Clock(), time.Local)
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, out)
	})).ServeHTTP)
}
