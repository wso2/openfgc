package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

type Principal struct {
	Subject string
	OrgID   string
	Scopes  []string
}
type Validator struct {
	cfg     config.AuthConfig
	client  *http.Client
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
	jwksURL string
}
type discovery struct {
	JWKSURI string `json:"jwks_uri"`
}
type jwks struct {
	Keys []jwk `json:"keys"`
}
type jwk struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewValidator(cfg config.AuthConfig) *Validator {
	return &Validator{cfg: cfg, client: &http.Client{Timeout: cfg.JWKSRefreshTimeout}, keys: map[string]*rsa.PublicKey{}}
}
func (v *Validator) Validate(ctx context.Context, raw string) (Principal, error) {
	token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok || !contains(v.cfg.AllowedAlgorithms, t.Method.Alg()) {
			return nil, fmt.Errorf("unsupported signing algorithm")
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("missing kid")
		}
		key, err := v.key(ctx, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	}, jwt.WithIssuer(v.cfg.IssuerURL), jwt.WithAudience(v.cfg.ResourceAudience), jwt.WithLeeway(v.cfg.ClockSkew), jwt.WithValidMethods(v.cfg.AllowedAlgorithms))
	if err != nil {
		return Principal{}, fmt.Errorf("parse and validate JWT: %w", err)
	}
	if !token.Valid {
		return Principal{}, fmt.Errorf("JWT is not valid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, fmt.Errorf("invalid claims")
	}
	sub, _ := claims.GetSubject()
	// sub, _ = claims["email"].(string) // Uncomment this to use email as sub
	sub = strings.TrimSpace(sub)
	org, _ := claims["org_id"].(string)
	org = strings.TrimSpace(org)
	if sub == "" || org == "" {
		return Principal{}, fmt.Errorf("missing identity claims")
	}
	if v.cfg.RequireAccessTokenType {
		if got, _ := claims[v.cfg.TokenTypeClaim].(string); got != v.cfg.AccessTokenType {
			return Principal{}, fmt.Errorf("wrong token type")
		}
	}
	scope, _ := claims["scope"].(string)
	return Principal{Subject: sub, OrgID: org, Scopes: strings.Fields(scope)}, nil
}
func (v *Validator) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Now().After(v.expires) || v.keys[kid] == nil {
		if err := v.refresh(ctx); err != nil {
			if k := v.keys[kid]; k != nil {
				return k, nil
			}
			return nil, err
		}
	}
	k := v.keys[kid]
	if k == nil {
		return nil, fmt.Errorf("unknown kid")
	}
	return k, nil
}
func (v *Validator) refresh(ctx context.Context) error {
	if v.jwksURL == "" {
		r, e := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(v.cfg.IssuerURL, "/")+"/.well-known/openid-configuration", nil)
		if e != nil {
			return e
		}
		resp, e := v.client.Do(r)
		if e != nil {
			return e
		}
		defer resp.Body.Close()
		var d discovery
		if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&d) != nil || d.JWKSURI == "" {
			return fmt.Errorf("discovery failed")
		}
		v.jwksURL = d.JWKSURI
	}
	r, e := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if e != nil {
		return e
	}
	resp, e := v.client.Do(r)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	var set jwks
	if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&set) != nil {
		return fmt.Errorf("jwks fetch failed")
	}
	next := map[string]*rsa.PublicKey{}
	for _, j := range set.Keys {
		if j.KTY != "RSA" || j.KID == "" {
			continue
		}
		n, e := base64.RawURLEncoding.DecodeString(j.N)
		if e != nil {
			continue
		}
		eb, e := base64.RawURLEncoding.DecodeString(j.E)
		if e != nil {
			continue
		}
		next[j.KID] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(eb).Int64())}
	}
	if len(next) == 0 {
		return fmt.Errorf("no usable jwks")
	}
	v.keys = next
	v.expires = time.Now().Add(v.cfg.JWKSTTL)
	return nil
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
