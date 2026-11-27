// The account's own API keys: made, listed, switched off and removed from
// its console. Every route takes the owner from the signed-in account, never
// from the request, so one account can't name another's keys.
package gui

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/yetone/magpie/internal/access"
)

func userKeyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/user/keys", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		u, _ := userOf(r)
		keys, err := access.UserKeys(u.ID)
		if err != nil {
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"keys": keys})
	})).ServeHTTP)
	mux.HandleFunc("POST /api/user/keys", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Name   string    `json:"name"`
			Models *[]string `json:"models"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		if in.Models == nil || len(*in.Models) == 0 {
			authFail(rw, http.StatusBadRequest, "choose at least one open model for this key")
			return
		}
		u, _ := userOf(r)
		k, secret, err := access.CreateUserKey(u.ID, in.Name, *in.Models)
		if err != nil {
			if errors.Is(err, access.ErrNoKey) {
				authFail(rw, http.StatusNotFound, err.Error())
				return
			}
			authFail(rw, http.StatusBadRequest, err.Error())
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(map[string]any{"key": k, "secret": secret})
	})).ServeHTTP)
	mux.HandleFunc("PATCH /api/user/keys/{id}", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		var in struct {
			Off    *bool     `json:"off"`
			Models *[]string `json:"models"`
		}
		if !authBody(rw, r, &in) {
			return
		}
		if in.Off == nil && in.Models == nil {
			authFail(rw, http.StatusBadRequest, "off says whether the key is off, or models says what it may use")
			return
		}
		if in.Models != nil && len(*in.Models) == 0 {
			authFail(rw, http.StatusBadRequest, "choose at least one open model for this key")
			return
		}
		u, _ := userOf(r)
		k, err := access.UpdateUserKey(u.ID, r.PathValue("id"), in.Off, in.Models)
		if err != nil {
			if errors.Is(err, access.ErrNoKey) {
				authFail(rw, http.StatusNotFound, "no such API key")
				return
			}
			if errors.Is(err, access.ErrBadModels) || errors.Is(err, access.ErrEmptyGrant) {
				authFail(rw, http.StatusBadRequest, err.Error())
				return
			}
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		rw.Header().Set("Cache-Control", "no-store")
		writeJSON(rw, map[string]any{"key": k})
	})).ServeHTTP)
	mux.HandleFunc("DELETE /api/user/keys/{id}", requireUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			authFail(rw, http.StatusForbidden, "another page sent this")
			return
		}
		u, _ := userOf(r)
		if err := access.DeleteUserKey(u.ID, r.PathValue("id")); err != nil {
			if errors.Is(err, access.ErrNoKey) {
				authFail(rw, http.StatusNotFound, "no such API key")
				return
			}
			authFail(rw, http.StatusInternalServerError, err.Error())
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	})).ServeHTTP)
}
