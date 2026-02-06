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

// Package context provides utilities for managing trace IDs (correlation IDs)
package context

import (
	"context"

	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/utils"
)

// GetTraceID retrieves the trace ID (correlation ID) from the context.
// If no trace ID exists, it generates a new UUID.
// This trace ID can be used to correlate logs, events, and operations across a request flow.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return utils.GenerateUUID()
	}

	if traceID, ok := ctx.Value(log.ContextKeyTraceID).(string); ok && traceID != "" {
		return traceID
	}

	return utils.GenerateUUID()
}

// WithTraceID adds a trace ID (correlation ID) to the context.
// Use this to propagate trace IDs through your application.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, log.ContextKeyTraceID, traceID)
}

// EnsureTraceID ensures a trace ID (correlation ID) exists in the context,
// generating one if needed. This is useful at entry points where you want to
// guarantee a trace ID is present for downstream operations.
func EnsureTraceID(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if traceID, ok := ctx.Value(log.ContextKeyTraceID).(string); !ok || traceID == "" {
		ctx = WithTraceID(ctx, utils.GenerateUUID())
	}

	return ctx
}
