package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

type testJWKS struct {
	mu   sync.RWMutex
	key  *rsa.PrivateKey
	kid  string
	fail bool
	hits atomic.Int64
}

func newTestValidator(t *testing.T) (*Validator, *testJWKS, func(jwt.MapClaims, string) string) {
	t.Helper()
	state := &testJWKS{kid: "key-1"}
	var err error
	state.key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.hits.Add(1)
		state.mu.RLock()
		defer state.mu.RUnlock()
		if state.fail {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/.well-known/openid-configuration" {
			_ = json.NewEncoder(w).Encode(discovery{JWKSURI: server.URL + "/keys"})
			return
		}
		if r.URL.Path == "/keys" {
			n := base64.RawURLEncoding.EncodeToString(state.key.PublicKey.N.Bytes())
			e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(state.key.PublicKey.E)).Bytes())
			_ = json.NewEncoder(w).Encode(jwks{Keys: []jwk{{KID: state.kid, KTY: "RSA", N: n, E: e}}})
			return
		}
		http.NotFound(w, r)
	}))
	cfg := config.AuthConfig{IssuerURL: server.URL, ResourceAudience: "consent-api", AllowedAlgorithms: []string{"RS256"}, JWKSTTL: time.Hour, JWKSRefreshTimeout: time.Second, ClockSkew: 0}
	sign := func(claims jwt.MapClaims, kid string) string {
		state.mu.RLock()
		key := state.key
		state.mu.RUnlock()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = kid
		raw, signErr := token.SignedString(key)
		if signErr != nil {
			t.Fatal(signErr)
		}
		return raw
	}
	return NewValidator(cfg), state, sign
}

func validClaims(issuer string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{"iss": issuer, "aud": "consent-api", "sub": "user-1", "org_id": "org-1", "scope": "portal:consents:read:self", "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix()}
}

func TestValidatorRejectsInvalidClaims(t *testing.T) {
	v, _, sign := newTestValidator(t)
	base := validClaims(v.cfg.IssuerURL)
	tests := []struct {
		name   string
		mutate func(jwt.MapClaims)
		alg    string
	}{
		{"wrong issuer", func(c jwt.MapClaims) { c["iss"] = "https://other" }, "RS256"},
		{"wrong audience", func(c jwt.MapClaims) { c["aud"] = "other" }, "RS256"},
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, "RS256"},
		{"not yet valid", func(c jwt.MapClaims) { c["nbf"] = time.Now().Add(time.Minute).Unix() }, "RS256"},
		{"missing sub", func(c jwt.MapClaims) { delete(c, "sub") }, "RS256"},
		{"missing org", func(c jwt.MapClaims) { delete(c, "org_id") }, "RS256"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := jwt.MapClaims{}
			for k, v := range base {
				c[k] = v
			}
			tt.mutate(c)
			if _, err := v.Validate(context.Background(), sign(c, "key-1")); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func TestValidatorRejectsInvalidSignatureAndTokenType(t *testing.T) {
	v, _, sign := newTestValidator(t)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims(v.cfg.IssuerURL))
	token.Header["kid"] = "key-1"
	raw, err := token.SignedString(other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Validate(context.Background(), raw); err == nil {
		t.Fatal("expected invalid signature")
	}
	v.cfg.RequireAccessTokenType = true
	v.cfg.TokenTypeClaim, v.cfg.AccessTokenType = "token_type", "access_token"
	claims := validClaims(v.cfg.IssuerURL)
	claims["token_type"] = "id_token"
	if _, err := v.Validate(context.Background(), sign(claims, "key-1")); err == nil {
		t.Fatal("expected token type rejection")
	}
}

func TestValidatorUsesCachedJWKS(t *testing.T) {
	v, state, sign := newTestValidator(t)
	raw := sign(validClaims(v.cfg.IssuerURL), "key-1")
	if _, err := v.Validate(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	first := state.hits.Load()
	if _, err := v.Validate(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	second := state.hits.Load()
	if second != first {
		t.Fatalf("expected cache hit without network refresh: %d -> %d", first, second)
	}
}

func TestValidatorParsesScopesAndRejectsUnknownAlgorithm(t *testing.T) {
	v, state, sign := newTestValidator(t)
	p, err := v.Validate(context.Background(), sign(validClaims(v.cfg.IssuerURL), "key-1"))
	if err != nil || len(p.Scopes) != 1 {
		t.Fatalf("expected valid scoped principal, got %#v, %v", p, err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims(v.cfg.IssuerURL))
	token.Header["kid"] = "key-1"
	raw, _ := token.SignedString([]byte("secret"))
	if _, err := v.Validate(context.Background(), raw); err == nil {
		t.Fatal("expected algorithm rejection")
	}
	_ = state
}

func TestValidatorRefreshesUnknownKIDAndKeepsLastKnownGoodKey(t *testing.T) {
	v, state, sign := newTestValidator(t)
	if _, err := v.Validate(context.Background(), sign(validClaims(v.cfg.IssuerURL), "key-1")); err != nil {
		t.Fatal(err)
	}
	state.mu.Lock()
	next, _ := rsa.GenerateKey(rand.Reader, 2048)
	state.key, state.kid = next, "key-2"
	state.mu.Unlock()
	if _, err := v.Validate(context.Background(), sign(validClaims(v.cfg.IssuerURL), "key-2")); err != nil {
		t.Fatalf("expected rotation refresh: %v", err)
	}
	state.mu.Lock()
	state.fail = true
	state.mu.Unlock()
	if _, err := v.Validate(context.Background(), sign(validClaims(v.cfg.IssuerURL), "key-2")); err != nil {
		t.Fatalf("expected cached key during refresh failure: %v", err)
	}
}
