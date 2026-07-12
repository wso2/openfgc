package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

type testIdP struct {
	server *httptest.Server
	key    *rsa.PrivateKey
}

func TestBearerAuthorizationCoversEveryAllowlistedAPIRoute(t *testing.T) {
	idp := newTestIdP(t)
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorizations":[],"purposes":[]}`))
	}))
	defer upstream.Close()
	bff := newBearerBFF(t, idp, upstream.URL)
	defer bff.Close()
	all := "portal:consents:read:any portal:consents:write:any portal:elements:read portal:elements:write portal:purposes:read portal:purposes:write"
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/consents"}, {http.MethodPost, "/api/consents"},
		{http.MethodGet, "/api/consents/attributes"}, {http.MethodPost, "/api/consents/validate"},
		{http.MethodGet, "/api/consents/id"}, {http.MethodPut, "/api/consents/id"}, {http.MethodPut, "/api/consents/id/revoke"},
		{http.MethodGet, "/api/consents/id/authorizations"}, {http.MethodPost, "/api/consents/id/authorizations"},
		{http.MethodGet, "/api/consents/id/authorizations/auth"}, {http.MethodPut, "/api/consents/id/authorizations/auth"},
		{http.MethodGet, "/api/consent-elements"}, {http.MethodPost, "/api/consent-elements"}, {http.MethodPost, "/api/consent-elements/validate"},
		{http.MethodGet, "/api/consent-elements/id"}, {http.MethodPut, "/api/consent-elements/id"}, {http.MethodDelete, "/api/consent-elements/id"},
		{http.MethodGet, "/api/consent-purposes"}, {http.MethodPost, "/api/consent-purposes"},
		{http.MethodGet, "/api/consent-purposes/id"}, {http.MethodPut, "/api/consent-purposes/id"}, {http.MethodDelete, "/api/consent-purposes/id"},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req, err := http.NewRequest(route.method, bff.URL+route.path+"?userIds=authorized&limit=10", strings.NewReader("{}"))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+idp.token(t, all))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
	if gotPath == "" {
		t.Fatal("expected protected routes to reach upstream")
	}
}

func TestBearerMeRouteOverwritesUserFilter(t *testing.T) {
	idp := newTestIdP(t)
	var gotUserIDs string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserIDs = r.URL.Query().Get("userIds")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer upstream.Close()
	bff := newBearerBFF(t, idp, upstream.URL)
	defer bff.Close()
	req, _ := http.NewRequest(http.MethodGet, bff.URL+"/me/consents?userIds=attacker", nil)
	req.Header.Set("Authorization", "Bearer "+idp.token(t, "portal:consents:read:self"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || gotUserIDs != "user@example.com" {
		t.Fatalf("expected self-scoped user ID, status=%d userIds=%q", resp.StatusCode, gotUserIDs)
	}
}

func TestBearerMeDetailApproveAndRevokeRoutes(t *testing.T) {
	idp := newTestIdP(t)
	const consentPath = "/api/v1/consents/550e8400-e29b-41d4-a716-446655440000"
	mutationCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == consentPath && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","clientId":"client-1","authorizations":[{"userId":"user@example.com"}],"purposes":[]}`))
			return
		}
		if r.Method == http.MethodPut {
			mutationCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()
	bff := newBearerBFF(t, idp, upstream.URL)
	defer bff.Close()
	tests := []struct{ name, method, path, body string }{
		{"detail", http.MethodGet, "/me/consents/550e8400-e29b-41d4-a716-446655440000", ""},
		{"approve", http.MethodPost, "/me/consents/550e8400-e29b-41d4-a716-446655440000/approve", "[]"},
		{"revoke", http.MethodPut, "/me/consents/550e8400-e29b-41d4-a716-446655440000/revoke", "{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, bff.URL+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+idp.token(t, "portal:consents:read:self portal:consents:write:self"))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
	if mutationCalls != 2 {
		t.Fatalf("expected approve and revoke mutations, got %d", mutationCalls)
	}
}

func newTestIdP(t *testing.T) *testIdP {
	t.Helper()
	idp := &testIdP{}
	var err error
	idp.key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	idp.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"jwks_uri": idp.server.URL + "/keys"})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kid": "test-key", "kty": "RSA", "n": base64.RawURLEncoding.EncodeToString(idp.key.PublicKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(idp.key.PublicKey.E)).Bytes())}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(idp.server.Close)
	return idp
}

func (idp *testIdP) token(t *testing.T, scopes string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{"iss": idp.server.URL, "aud": "consent-api", "sub": "user@example.com", "org_id": "ORG-001", "scope": scopes, "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	raw, err := token.SignedString(idp.key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newBearerBFF(t *testing.T, idp *testIdP, upstreamURL string) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Auth.IssuerURL = idp.server.URL
	cfg.Auth.ResourceAudience = "consent-api"
	cfg.Proxy.OpenFGCAPIURL = upstreamURL
	cfg.Proxy.PlaceholderModeEnabled = false
	cfg.Proxy.PlaceholderUserID = ""
	cfg.Proxy.PlaceholderOrgID = ""
	cfg.Proxy.PlaceholderClientID = ""
	cfg.CORS.AllowedOrigins = []string{"https://portal.example.com"}
	h, err := newIntegrationHandler(*cfg)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(h)
}

func TestBearerAuthenticationAndScopeIntegration(t *testing.T) {
	idp := newTestIdP(t)
	gotOrg := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("org-id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()
	bff := newBearerBFF(t, idp, upstream.URL)
	defer bff.Close()
	request := func(token string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, bff.URL+"/api/consents", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := request("")
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("expected 401 bearer challenge, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp = request(idp.token(t, "portal:consents:read:self"))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp = request(idp.token(t, "portal:consents:read:any"))
	if resp.StatusCode != http.StatusOK || gotOrg != "ORG-001" {
		t.Fatalf("expected authorized org-injected request, status=%d org=%q", resp.StatusCode, gotOrg)
	}
	_ = resp.Body.Close()
}

func TestCORSPreflightAllowsAuthorization(t *testing.T) {
	idp := newTestIdP(t)
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	bff := newBearerBFF(t, idp, upstream.URL)
	defer bff.Close()
	req, _ := http.NewRequest(http.MethodOptions, bff.URL+"/api/consents", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Headers") == "" {
		t.Fatalf("unexpected preflight response: %d", resp.StatusCode)
	}
}
