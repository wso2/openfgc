package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	systemcontext "github.com/wso2/openfgc/portal/backend/internal/system/context"
)

type identityKey struct{}

// IdentityOptions controls the explicit local/test placeholder mode.
type IdentityOptions struct {
	PlaceholderModeEnabled bool
	PlaceholderUserID      string
	PlaceholderOrgID       string
}

// Authenticate validates bearer credentials, or derives the same identity from
// explicitly enabled local/test placeholders. Placeholder mode is never a fallback.
func Authenticate(next http.Handler, validator *auth.Validator, opts IdentityOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if opts.PlaceholderModeEnabled {
			userID, orgID := strings.TrimSpace(opts.PlaceholderUserID), strings.TrimSpace(opts.PlaceholderOrgID)
			if userID == "" || orgID == "" {
				writeAuthError(w, http.StatusServiceUnavailable, "PLACEHOLDER_UNAVAILABLE", "placeholder identity unavailable")
				return
			}
			ctx := systemcontext.WithUserIdentity(r.Context(), systemcontext.UserIdentity{UserID: userID, OrgID: orgID})
			ctx = contextWithPlaceholder(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		raw, ok := bearerToken(r)
		if !ok {
			slog.Default().Warn("bearer authentication rejected", "reason", "missing or malformed credentials", "method", r.Method, "path", r.URL.Path)
			writeUnauthorized(w)
			return
		}
		principal, err := validator.Validate(r.Context(), raw)
		if err != nil {
			slog.Default().Warn("bearer authentication rejected", "reason", err.Error(), "method", r.Method, "path", r.URL.Path)
			writeUnauthorized(w)
			return
		}
		slog.Default().Debug("bearer authentication accepted", "subject", principal.Subject, "org_id", principal.OrgID, "scope_count", len(principal.Scopes), "method", r.Method, "path", r.URL.Path)
		ctx := systemcontext.WithUserIdentity(r.Context(), systemcontext.UserIdentity{UserID: principal.Subject, OrgID: principal.OrgID})
		ctx = contextWithPrincipal(ctx, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScope requires a scope from a validated bearer token. Explicit local/test
// placeholder mode bypasses scope checks because it has no token to supply scopes.
func RequireScope(next http.Handler, scope func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPlaceholder(r) {
			next.ServeHTTP(w, r)
			return
		}
		required := scope(r)
		principal, ok := principalFromContext(r.Context())
		if required == "" || !ok || !hasScope(principal.Scopes, required) {
			slog.Default().Warn("route authorization rejected", "required_scope", required, "method", r.Method, "path", r.URL.Path)
			writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "insufficient scope")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) (string, bool) {
	if r.URL.Query().Get("access_token") != "" || r.Header.Get("Cookie") != "" {
		return "", false
	}
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(userIDErrorResponse{Code: code, Message: message})
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}
	return false
}

func contextWithPrincipal(ctx context.Context, principal auth.Principal) context.Context {
	return context.WithValue(ctx, identityKey{}, principal)
}

func principalFromContext(ctx context.Context) (auth.Principal, bool) {
	principal, ok := ctx.Value(identityKey{}).(auth.Principal)
	return principal, ok
}

func contextWithPlaceholder(ctx context.Context) context.Context {
	return context.WithValue(ctx, identityKey{}, true)
}

func isPlaceholder(r *http.Request) bool {
	placeholder, _ := r.Context().Value(identityKey{}).(bool)
	return placeholder
}
