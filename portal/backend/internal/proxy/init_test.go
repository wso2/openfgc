package proxy

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAPIScopePolicy(t *testing.T) {
	tests := []struct{ method, path, want string }{
		{http.MethodGet, "/api/consents", "portal:consents:read:any"},
		{http.MethodPost, "/api/consents", "portal:consents:write:any"},
		{http.MethodGet, "/api/consent-elements", "portal:elements:read"},
		{http.MethodDelete, "/api/consent-elements/id", "portal:elements:write"},
		{http.MethodGet, "/api/consent-purposes/id", "portal:purposes:read"},
		{http.MethodPut, "/api/consent-purposes/id", "portal:purposes:write"},
	}
	for _, tt := range tests {
		if got := apiScope(&http.Request{Method: tt.method, URL: mustURL(tt.path)}); got != tt.want {
			t.Errorf("%s %s: got %q, want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func mustURL(path string) *url.URL { u, _ := url.Parse(path); return u }
