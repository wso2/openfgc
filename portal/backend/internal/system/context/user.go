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

// Package context provides request-scoped values shared across modules.
package context

import (
	"context"
	"strings"
)

type userIDKey struct{}
type userIdentityKey struct{}

// UserIdentity is the only user identity representation exposed to application code.
// It is derived from validated token claims or explicit local/test placeholders.
type UserIdentity struct {
	UserID string
	OrgID  string
}

// WithUserIdentity stores a normalized user identity in request context.
func WithUserIdentity(ctx context.Context, identity UserIdentity) context.Context {
	identity.UserID = strings.TrimSpace(identity.UserID)
	identity.OrgID = strings.TrimSpace(identity.OrgID)
	return context.WithValue(ctx, userIdentityKey{}, identity)
}

// UserIdentityFromContext returns the identity resolved by authentication middleware.
func UserIdentityFromContext(ctx context.Context) (UserIdentity, bool) {
	if ctx == nil {
		return UserIdentity{}, false
	}
	identity, ok := ctx.Value(userIdentityKey{}).(UserIdentity)
	if !ok || strings.TrimSpace(identity.UserID) == "" || strings.TrimSpace(identity.OrgID) == "" {
		return UserIdentity{}, false
	}
	identity.UserID = strings.TrimSpace(identity.UserID)
	identity.OrgID = strings.TrimSpace(identity.OrgID)
	return identity, true
}

// WithUserID stores the effective user ID in request context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext returns the effective user ID previously resolved by middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	if identity, ok := UserIdentityFromContext(ctx); ok {
		return identity.UserID, true
	}
	if ctx == nil {
		return "", false
	}

	value, ok := ctx.Value(userIDKey{}).(string)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	return value, true
}
