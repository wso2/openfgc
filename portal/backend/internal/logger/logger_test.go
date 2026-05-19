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

package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewLevelMapping(t *testing.T) {
	tests := []struct {
		name  string
		level string
		debug bool
		info  bool
		warn  bool
		error bool
	}{
		{
			name:  "debug enables all levels",
			level: "debug",
			debug: true,
			info:  true,
			warn:  true,
			error: true,
		},
		{
			name:  "info enables info and above",
			level: "info",
			debug: false,
			info:  true,
			warn:  true,
			error: true,
		},
		{
			name:  "warn enables warn and error",
			level: "warn",
			debug: false,
			info:  false,
			warn:  true,
			error: true,
		},
		{
			name:  "error enables only error",
			level: "error",
			debug: false,
			info:  false,
			warn:  false,
			error: true,
		},
		{
			name:  "empty defaults to info",
			level: "",
			debug: false,
			info:  true,
			warn:  true,
			error: true,
		},
		{
			name:  "invalid defaults to info",
			level: "invalid",
			debug: false,
			info:  true,
			warn:  true,
			error: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.level)
			if log == nil {
				t.Fatal("expected non-nil logger")
			}

			assertEnabled(t, log, slog.LevelDebug, tt.debug)
			assertEnabled(t, log, slog.LevelInfo, tt.info)
			assertEnabled(t, log, slog.LevelWarn, tt.warn)
			assertEnabled(t, log, slog.LevelError, tt.error)
		})
	}
}

func TestNewLevelCaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		level string
		check slog.Level
		want  bool
	}{
		{name: "uppercase debug", level: "DEBUG", check: slog.LevelDebug, want: true},
		{name: "mixed warn", level: "WaRn", check: slog.LevelInfo, want: false},
		{name: "uppercase error", level: "ERROR", check: slog.LevelWarn, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.level)
			if log == nil {
				t.Fatal("expected non-nil logger")
			}

			got := log.Enabled(context.Background(), tt.check)
			if got != tt.want {
				t.Fatalf("expected enabled(%v)=%v, got %v", tt.check, tt.want, got)
			}
		})
	}
}

func assertEnabled(t *testing.T, log *slog.Logger, level slog.Level, want bool) {
	t.Helper()

	got := log.Enabled(context.Background(), level)
	if got != want {
		t.Fatalf("expected enabled(%v)=%v, got %v", level, want, got)
	}
}
