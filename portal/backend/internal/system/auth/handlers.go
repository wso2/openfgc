/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package auth

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Login starts the OIDC authorization-code flow.
func (m *Manager) Login(w http.ResponseWriter, r *http.Request) {
	if !m.cfg.Enabled {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	state, verifier, err := newLoginTransaction()
	if err != nil {
		m.log.Error("OIDC login transaction generation failed")
		writeError(w, http.StatusInternalServerError, "LOGIN_FAILED", "login failed")
		return
	}
	m.setLoginTransactionCookies(w, state, verifier)
	authorizationURL := m.oauthConfig.AuthCodeURL(
		state,
		pkceAuthorizationOption(verifier),
		oauth2.SetAuthURLParam("audience", m.cfg.ResourceAudience),
	)
	http.Redirect(w, r, authorizationURL, http.StatusFound)
}

// Callback exchanges an authorization code and issues split-token cookies.
func (m *Manager) Callback(w http.ResponseWriter, r *http.Request) {
	if !m.cfg.Enabled {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	states := r.URL.Query()["state"]
	expectedState, stateErr := exactlyOneCookie(r, m.cfg.OAuthStateCookie)
	verifier, verifierErr := exactlyOneCookie(r, m.cfg.PKCEVerifierCookie)
	m.clearLoginTransactionCookies(w)
	if len(states) != 1 || !statesMatch(expectedState, states[0]) || stateErr != nil ||
		verifierErr != nil || !validPKCEVerifier(verifier) {
		m.callbackFailure(w, r, "invalid login transaction")
		return
	}
	codes := r.URL.Query()["code"]
	if len(codes) != 1 || strings.TrimSpace(codes[0]) == "" {
		m.callbackFailure(w, r, "missing authorization code")
		return
	}
	ctx, cancel := timeBoundOIDCContext(r, m.httpClient, m.cfg.HTTPTimeout)
	defer cancel()
	token, err := m.oauthConfig.Exchange(ctx, codes[0], oauth2.VerifierOption(verifier))
	if err != nil {
		m.callbackFailure(w, r, "token exchange failed")
		return
	}
	idToken, _ := token.Extra("id_token").(string)
	if token.AccessToken == "" || token.RefreshToken == "" || idToken == "" {
		m.callbackFailure(w, r, "token response incomplete")
		return
	}
	_, accessExpiry, err := m.validateAccessToken(r.Context(), token.AccessToken)
	if err != nil {
		m.callbackFailure(w, r, "access token validation failed")
		return
	}
	idExpiry, err := m.validateIDToken(r.Context(), idToken)
	if err != nil {
		m.callbackFailure(w, r, "id token validation failed")
		return
	}
	refreshExpiry := time.Now().Add(time.Duration(m.cfg.RefreshCookieMaxAgeSeconds) * time.Second)
	if err := m.validateCookieSet(token.AccessToken, token.RefreshToken, idToken); err != nil {
		m.callbackFailure(w, r, "token cookie limits exceeded")
		return
	}
	_ = m.setSplitCookies(w, token.AccessToken, m.cfg.AccessTokenPart1Cookie, m.cfg.AccessTokenPart2Cookie, true, accessExpiry)
	_ = m.setSplitCookies(w, token.RefreshToken, m.cfg.RefreshTokenPart1Cookie, m.cfg.RefreshTokenPart2Cookie, true, refreshExpiry)
	_ = m.setSplitCookies(w, idToken, m.cfg.IDTokenPart1Cookie, m.cfg.IDTokenPart2Cookie, false, idExpiry)
	http.Redirect(w, r, m.cfg.PortalURL, http.StatusFound)
}

// Refresh exchanges the reconstructed refresh token and rotates token cookies.
func (m *Manager) Refresh(w http.ResponseWriter, r *http.Request) {
	if !m.cfg.Enabled {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "form-encoded refresh token required")
		return
	}
	// Allow for URL percent-encoding expansion, then enforce the decoded part limit below.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(m.cfg.MaxTokenPartBytes*3+64)))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid refresh request")
		return
	}
	values, err := url.ParseQuery(string(body))
	parts := values["refresh_token"]
	if err != nil || len(parts) != 1 || parts[0] == "" || len(values) != 1 {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh credentials")
		return
	}
	part2, err := exactlyOneCookie(r, m.cfg.RefreshTokenPart2Cookie)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh credentials")
		return
	}
	refreshToken, err := reconstructToken(parts[0], part2, m.cfg)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh credentials")
		return
	}
	ctx, cancel := timeBoundOIDCContext(r, m.httpClient, m.cfg.RefreshTimeout)
	defer cancel()
	source := m.oauthConfig.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken, Expiry: time.Now().Add(-time.Hour)})
	newToken, err := source.Token()
	if err != nil || newToken.AccessToken == "" {
		m.clearTokenCookies(w)
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session refresh failed")
		return
	}
	_, accessExpiry, err := m.validateAccessToken(r.Context(), newToken.AccessToken)
	if err != nil {
		m.clearTokenCookies(w)
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session refresh failed")
		return
	}
	idToken, _ := newToken.Extra("id_token").(string)
	var idExpiry time.Time
	if idToken != "" {
		idExpiry, err = m.validateIDToken(r.Context(), idToken)
		if err != nil {
			m.clearTokenCookies(w)
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session refresh failed")
			return
		}
	}
	if _, _, err := splitToken(newToken.AccessToken, m.cfg.MaxTokenPartBytes); err != nil {
		m.clearTokenCookies(w)
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session refresh failed")
		return
	}
	_ = m.setSplitCookies(w, newToken.AccessToken, m.cfg.AccessTokenPart1Cookie, m.cfg.AccessTokenPart2Cookie, true, accessExpiry)
	if newToken.RefreshToken != "" && newToken.RefreshToken != refreshToken {
		if _, _, err := splitToken(newToken.RefreshToken, m.cfg.MaxTokenPartBytes); err != nil {
			m.clearTokenCookies(w)
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session refresh failed")
			return
		}
		refreshExpiry := time.Now().Add(time.Duration(m.cfg.RefreshCookieMaxAgeSeconds) * time.Second)
		_ = m.setSplitCookies(w, newToken.RefreshToken, m.cfg.RefreshTokenPart1Cookie, m.cfg.RefreshTokenPart2Cookie, true, refreshExpiry)
	}
	if idToken != "" {
		if _, _, err := splitToken(idToken, m.cfg.MaxTokenPartBytes); err != nil {
			m.clearTokenCookies(w)
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session refresh failed")
			return
		}
		_ = m.setSplitCookies(w, idToken, m.cfg.IDTokenPart1Cookie, m.cfg.IDTokenPart2Cookie, false, idExpiry)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Logout clears local cookies and returns the IdP end-session URL when available.
func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) {
	m.clearTokenCookies(w)
	logoutURL := m.cfg.PostLogoutRedirectURI
	if m.endSessionEndpoint != "" {
		if parsed, err := url.Parse(m.endSessionEndpoint); err == nil {
			query := parsed.Query()
			query.Set("post_logout_redirect_uri", m.cfg.PostLogoutRedirectURI)
			if idToken, tokenErr := joinCookieToken(r, m.cfg.IDTokenPart1Cookie, m.cfg.IDTokenPart2Cookie, m.cfg); tokenErr == nil {
				if _, verifyErr := m.validateIDToken(r.Context(), idToken); verifyErr == nil {
					query.Set("id_token_hint", idToken)
				}
			}
			parsed.RawQuery = query.Encode()
			logoutURL = parsed.String()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"logoutUrl": logoutURL})
}

func (m *Manager) callbackFailure(w http.ResponseWriter, r *http.Request, reason string) {
	m.log.Warn("OIDC callback rejected", "reason", reason)
	target, err := url.Parse(m.cfg.PortalURL)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "LOGIN_FAILED", "login failed")
		return
	}
	query := target.Query()
	query.Set("auth_error", "login_failed")
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (m *Manager) validateCookieSet(accessToken, refreshToken, idToken string) error {
	for _, token := range []string{accessToken, refreshToken, idToken} {
		if len(token) > m.cfg.MaxReconstructedTokenBytes {
			return errInvalidCredentials
		}
		if _, _, err := splitToken(token, m.cfg.MaxTokenPartBytes); err != nil {
			return err
		}
	}
	return nil
}

func timeBoundOIDCContext(r *http.Request, client *http.Client, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	return oidc.ClientContext(ctx, client), cancel
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}
