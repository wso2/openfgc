package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	systemcontext "github.com/wso2/openfgc/portal/backend/internal/system/context"
)

func TestAuthenticateRejectsInvalidBearerHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := Authenticate(next, nil, IdentityOptions{})
	tests := []struct {
		name    string
		headers []string
	}{
		{"missing", nil}, {"malformed", []string{"Basic abc"}}, {"duplicate", []string{"Bearer one", "Bearer two"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/me/consents", nil)
			for _, value := range tt.headers {
				r.Header.Add("Authorization", value)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", w.Code)
			}
			if got := w.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Fatalf("expected WWW-Authenticate Bearer, got %q", got)
			}
		})
	}
}

func TestRequireScopeReturnsForbiddenWithoutValidatedPrincipal(t *testing.T) {
	h := RequireScope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), func(*http.Request) string { return "portal:consents:read:self" })
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me/consents", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestPlaceholderAuthenticationWritesUserIdentity(t *testing.T) {
	var got systemcontext.UserIdentity
	h := Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		got, ok = systemcontext.UserIdentityFromContext(r.Context())
		if !ok {
			t.Fatal("missing identity")
		}
		w.WriteHeader(http.StatusNoContent)
	}), nil, IdentityOptions{PlaceholderModeEnabled: true, PlaceholderUserID: "user-1", PlaceholderOrgID: "org-1"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me/consents", nil))
	if w.Code != http.StatusNoContent || got.UserID != "user-1" || got.OrgID != "org-1" {
		t.Fatalf("unexpected result: status=%d identity=%#v", w.Code, got)
	}
}
