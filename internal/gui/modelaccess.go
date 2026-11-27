// The platform's model access: the admin sees every model the Gateway
// serves and opens or closes it, names it and writes its description; a
// user sees only the open ones in the model plaza. Both read
// internal/modelaccess, the same rule the Gateway and the user's keys
// enforce, so what is shown is exactly what may be called.
package gui

import (
	"net"
	"net/http"
	"strings"

	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/modelaccess"
)

// gatewayBaseFor is the address a user's clients call: the public gateway
// URL when one is set, else this deployment's gateway on the host this page
// was reached by. The page and the gateway are often on different ports
// (3430 and 3425), so the page's own origin would be the wrong example.
func gatewayBaseFor(r *http.Request) string {
	if u := gateway.PublicURL(); u != "" {
		return strings.TrimSuffix(u, "/")
	}
	scheme := gateway.Scheme()
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.Trim(host, "[]")
	}
	addrHost, port, err := net.SplitHostPort(gateway.Addr())
	if err != nil || port == "" {
		return gateway.URL()
	}
	if host == "" {
		host = addrHost
	}
	// a gateway bound to one non-loopback address can't be reached at the
	// page's loopback address; the address it listens on is the one
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if aip := net.ParseIP(addrHost); aip != nil && !aip.IsLoopback() && !aip.IsUnspecified() {
			host = addrHost
		}
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

func modelAccessRoutes(mux *http.ServeMux) {
	// the user's plaza: open models only, with no admin fields
	mux.HandleFunc("GET /api/user/models", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"models": modelaccess.OpenModels(), "gateway": gatewayBaseFor(r)})
	})).ServeHTTP)

	// the administrator's catalog: every model, open or closed. The
	// management page's own address is guarded by its page key (or the
	// app's local window); the portal's is guarded by the signed-in
	// administrator there.
	mux.HandleFunc("GET /api/model-access", modelAccessList)
	mux.HandleFunc("POST /api/model-access", modelAccessUpdate)
	mux.HandleFunc("POST /api/model-access/groups-only", modelAccessGroupsOnly)

	mux.HandleFunc("GET /api/admin/model-access", requireAdmin(http.HandlerFunc(modelAccessList)).ServeHTTP)
	mux.HandleFunc("POST /api/admin/model-access", requireAdmin(http.HandlerFunc(modelAccessUpdate)).ServeHTTP)
	mux.HandleFunc("POST /api/admin/model-access/groups-only", requireAdmin(http.HandlerFunc(modelAccessGroupsOnly)).ServeHTTP)
}

func modelAccessList(rw http.ResponseWriter, _ *http.Request) {
	models := modelaccess.AllModels()
	open := 0
	for _, m := range models {
		if m.Open {
			open++
		}
	}
	rw.Header().Set("Cache-Control", "no-store")
	writeJSON(rw, map[string]any{"models": models, "total": len(models), "open": open, "groupsOnly": modelaccess.GroupsOnly()})
}

// modelAccessUpdate opens, closes, names or describes one model.
func modelAccessUpdate(rw http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		authFail(rw, http.StatusForbidden, "another page sent this")
		return
	}
	var in struct {
		ID          string  `json:"id"`
		Open        *bool   `json:"open"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !authBody(rw, r, &in) {
		return
	}
	if in.Open == nil && in.Name == nil && in.Description == nil {
		authFail(rw, http.StatusBadRequest, "give open, name or description")
		return
	}
	m, err := modelaccess.Update(in.ID, in.Open, in.Name, in.Description)
	if err != nil {
		authFail(rw, http.StatusBadRequest, err.Error())
		return
	}
	rw.Header().Set("Cache-Control", "no-store")
	writeJSON(rw, map[string]any{"model": m})
}

// modelAccessGroupsOnly sets what the plaza and new user keys are held to:
// every open model, or open routing groups alone.
func modelAccessGroupsOnly(rw http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		authFail(rw, http.StatusForbidden, "another page sent this")
		return
	}
	var in struct {
		GroupsOnly *bool `json:"groupsOnly"`
	}
	if !authBody(rw, r, &in) {
		return
	}
	if in.GroupsOnly == nil {
		authFail(rw, http.StatusBadRequest, "groupsOnly says whether the plaza shows routing groups alone")
		return
	}
	if err := modelaccess.SetGroupsOnly(*in.GroupsOnly); err != nil {
		authFail(rw, http.StatusInternalServerError, err.Error())
		return
	}
	rw.Header().Set("Cache-Control", "no-store")
	writeJSON(rw, map[string]any{"groupsOnly": *in.GroupsOnly})
}
