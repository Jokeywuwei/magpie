// The platform's account endpoints: sign-up, sign-in, sign-out, who this
// browser is, and the admin's management of accounts. They are served
// outside `magpie web`'s page key (web.go's public paths), because a user
// signs in with an account, not with the admin's link; each route here
// checks the account itself, and the admin ones check its role.
package gui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/identity"
)

// SessionCookie is the browser's signed-in token. It is HttpOnly,
// SameSite=Lax and Secure when the page is HTTPS, here or at a proxy that
// says so; the gateway keys are unrelated cookies.
const SessionCookie = "magpie_session"

// secureRequest says the browser reached this page over HTTPS, directly or
// through the proxy MAGPIE_PUBLIC_URL names. A forwarding header on its own
// is not trusted: any caller can write X-Forwarded-Proto, and a page served
// plainly over HTTP would then set a Secure cookie the browser drops (a
// self-inflicted sign-out), or a proxy could be impersonated. When the
// deployment has configured an HTTPS public address, a request whose Host is
// that address and whose trusted proxy says https gets the Secure cookie; a
// direct TLS connection always does.
func secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	u, err := url.Parse(gateway.PublicURL())
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Host), u.Host)
}

type authUserKey struct{}

// authOrigins turns away a state-changing request another site sent with
// this browser's cookie: SameSite=Lax keeps the cookie off a cross-site
// POST, and this checks the Origin of one that arrived without it.
var authOrigins = http.NewCrossOriginProtection()

func sameOrigin(r *http.Request) bool { return authOrigins.Check(r) == nil }

func userOf(r *http.Request) (identity.User, bool) {
	u, ok := r.Context().Value(authUserKey{}).(identity.User)
	return u, ok
}

// sessionUser is the account this request's session cookie signs in. A
// browser may carry more than one cookie of the name (one from an older
// run); any that still signs in is the browser's.
func sessionUser(r *http.Request) (identity.User, bool) {
	for _, c := range r.CookiesNamed(SessionCookie) {
		if u, err := identity.SessionUser(c.Value); err == nil {
			return u, true
		}
	}
	return identity.User{}, false
}

// sessionToken is the token in the request's cookies, if one was sent.
func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(SessionCookie); err == nil {
		return c.Value
	}
	return ""
}

func setSession(rw http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	c := &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(identity.SessionTTL / time.Second),
		Secure:   secureRequest(r),
	}
	rw.Header().Add("Set-Cookie", c.String()+"; Priority=High")
}

func clearSession(rw http.ResponseWriter, r *http.Request) {
	http.SetCookie(rw, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0), Secure: secureRequest(r)})
}

// requireUser answers 401 without a sign-in and passes a signed-in account
// on, in the request's context.
func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		u, ok := sessionUser(r)
		if !ok {
			authFail(rw, http.StatusUnauthorized, "sign in first")
			return
		}
		next.ServeHTTP(rw, r.WithContext(context.WithValue(r.Context(), authUserKey{}, u)))
	})
}

// requireAdmin is requireUser and the admin role: 401 without a sign-in,
// 403 for a signed-in user.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		u, ok := sessionUser(r)
		if !ok {
			authFail(rw, http.StatusUnauthorized, "sign in as an admin first")
			return
		}
		if u.Role != identity.RoleAdmin {
			authFail(rw, http.StatusForbidden, "an admin only")
			return
		}
		next.ServeHTTP(rw, r.WithContext(context.WithValue(r.Context(), authUserKey{}, u)))
	})
}

func authFail(rw http.ResponseWriter, code int, why string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(map[string]string{"error": why})
}

func authStatus(err error) (int, string) {
	switch {
	case errors.Is(err, identity.ErrEmailTaken):
		return http.StatusConflict, err.Error()
	case errors.Is(err, identity.ErrBadLogin):
		return http.StatusUnauthorized, err.Error()
	case errors.Is(err, identity.ErrDisabled):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, identity.ErrRegistrationClosed):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, identity.ErrNoAdmin):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, identity.ErrLastAdmin):
		return http.StatusConflict, err.Error()
	case errors.Is(err, identity.ErrNotFound):
		return http.StatusNotFound, err.Error()
	}
	return http.StatusBadRequest, err.Error()
}

// authBody reads a JSON body, at most 64 KiB, into v.
func authBody(rw http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(rw, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		authFail(rw, http.StatusBadRequest, "the request isn't the JSON expected")
		return false
	}
	return true
}

func authUser(rw http.ResponseWriter, u identity.User, code int) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(map[string]any{"user": u.Account()})
}

// authRoutes are the public account routes. Sign-in is open; sign-up is
// open only after an administrator exists and has turned registration on,
// and the first administrator is made with `magpie user add --role admin`.
// The rest check the cookie.
func authRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		open, err := identity.RegistrationOpen()
		if err != nil {
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		if !open {
			authFail(rw, http.StatusForbidden, identity.ErrRegistrationClosed.Error())
			return
		}
		u, err := identity.Register(in.Email, in.Name, in.Password)
		if err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		token, expires, err := identity.NewSession(u.ID)
		if err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		setSession(rw, r, token, expires)
		authUser(rw, u, http.StatusCreated)
	})
	mux.HandleFunc("POST /api/auth/login", func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			// Email is what was typed: an email or the part before @. The
			// field keeps its name for clients that already send it.
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		login, ip := strings.ToLower(strings.TrimSpace(in.Email)), loginIP(r)
		if wait := loginAttempts.retry(login, ip); wait > 0 {
			// the same answer for a known and an unknown account: a limit
			// says nothing about which accounts exist
			rw.Header().Set("Retry-After", fmt.Sprint(max(1, int(wait.Seconds()+0.999))))
			authFail(rw, http.StatusTooManyRequests, "too many sign-in attempts: try again later")
			return
		}
		u, err := identity.Authenticate(in.Email, in.Password)
		if err != nil {
			if errors.Is(err, identity.ErrBadLogin) {
				loginAttempts.fail(login, ip)
			}
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		loginAttempts.success(login, ip)
		token, expires, err := identity.NewSession(u.ID)
		if err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		setSession(rw, r, token, expires)
		authUser(rw, u, http.StatusOK)
	})
	mux.HandleFunc("POST /api/auth/logout", func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		if err := identity.EndSession(sessionToken(r)); err != nil {
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		clearSession(rw, r)
		rw.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/auth/me", func(rw http.ResponseWriter, r *http.Request) {
		u, ok := sessionUser(r)
		if !ok {
			authFail(rw, http.StatusUnauthorized, "not signed in")
			return
		}
		authUser(rw, u, http.StatusOK)
	})
}

// accountAdminRoutes manage the platform's accounts. Every one needs an admin's
// session: the page key alone is not an account, and a user's session is
// 403.
func accountAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/users", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		users, err := identity.List()
		if err != nil {
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]any{"users": users})
	})).ServeHTTP)
	mux.HandleFunc("POST /api/admin/users", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Email    string        `json:"email"`
			Name     string        `json:"name"`
			Password string        `json:"password"`
			Role     identity.Role `json:"role"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		u, err := identity.Add(in.Email, in.Name, in.Password, in.Role)
		if err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		authUser(rw, u, http.StatusCreated)
	})).ServeHTTP)
	mux.HandleFunc("PATCH /api/admin/users/{id}", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Role     *identity.Role `json:"role"`
			Disabled *bool          `json:"disabled"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		id := r.PathValue("id")
		if in.Role != nil {
			if _, err := identity.SetRole(id, *in.Role); err != nil {
				code, why := authStatus(err)
				authFail(rw, code, why)
				return
			}
		}
		if in.Disabled != nil {
			if _, err := identity.SetDisabled(id, *in.Disabled); err != nil {
				code, why := authStatus(err)
				authFail(rw, code, why)
				return
			}
		}
		u, err := identity.ByID(id)
		if err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		authUser(rw, u, http.StatusOK)
	})).ServeHTTP)
	mux.HandleFunc("POST /api/admin/users/{id}/password", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Password string `json:"password"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		if err := identity.SetPassword(r.PathValue("id"), in.Password); err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	})).ServeHTTP)
	mux.HandleFunc("DELETE /api/admin/users/{id}", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		id := r.PathValue("id")
		if err := identity.Remove(id); err != nil {
			code, why := authStatus(err)
			authFail(rw, code, why)
			return
		}
		// the account's API keys go with it; one left behind is refused
		// anyway (the gateway checks its owner), but it is no longer kept
		_, _ = access.DeleteUserKeys(id)
		rw.WriteHeader(http.StatusNoContent)
	})).ServeHTTP)
	mux.HandleFunc("GET /api/admin/registration", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		open, err := identity.RegistrationOpen()
		if err != nil {
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"open": open})
	})).ServeHTTP)
	mux.HandleFunc("POST /api/admin/registration", requireAdmin(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Open *bool `json:"open"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		if in.Open == nil {
			authFail(rw, http.StatusBadRequest, "open says whether registration is open")
			return
		}
		if err := identity.SetRegistrationOpen(*in.Open); err != nil {
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"open": *in.Open})
	})).ServeHTTP)
}

// publicAPI is a path served without magpie web's page key: a user's own
// sign-in and the admin account routes, each of which checks its own
// account. Everything else stays behind the page's key.
func publicAPI(path string) bool {
	for _, p := range []string{"/api/auth", "/api/admin", "/api/user"} {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}
