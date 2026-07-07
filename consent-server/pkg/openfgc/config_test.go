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

package openfgc

import "testing"

func TestToInternalPartialStatusOverride(t *testing.T) {
	cfg := Config{
		StatusMappings:     &StatusMappings{Active: "LIVE"},
		AuthStatusMappings: &AuthStatusMappings{Approved: "OK"},
	}

	internal := cfg.toInternal()
	sm := internal.Consent.StatusMappings
	asm := internal.Consent.AuthStatusMappings

	// Overridden field takes the new value.
	if sm.ActiveStatus != "LIVE" {
		t.Errorf("ActiveStatus = %q, want LIVE", sm.ActiveStatus)
	}
	if asm.ApprovedState != "OK" {
		t.Errorf("ApprovedState = %q, want OK", asm.ApprovedState)
	}

	// Unspecified fields fall back to defaults, not empty strings.
	def := defaultStatusMappings()
	if sm.ExpiredStatus != def.ExpiredStatus || sm.RevokedStatus != def.RevokedStatus ||
		sm.CreatedStatus != def.CreatedStatus || sm.RejectedStatus != def.RejectedStatus {
		t.Errorf("partial StatusMappings override zeroed defaults: %+v", sm)
	}
	defAuth := defaultAuthStatusMappings()
	if asm.RejectedState != defAuth.RejectedState || asm.CreatedState != defAuth.CreatedState ||
		asm.SystemExpiredState != defAuth.SystemExpiredState || asm.SystemRevokedState != defAuth.SystemRevokedState {
		t.Errorf("partial AuthStatusMappings override zeroed defaults: %+v", asm)
	}
}

func TestToInternalHistoryAndLogLevelPassthrough(t *testing.T) {
	// Defaults: history off, no log level override.
	def := Config{}.toInternal()
	if def.Consent.History.Enabled {
		t.Errorf("History.Enabled = true by default, want false")
	}
	if def.Logging.Level != "" {
		t.Errorf("Logging.Level = %q by default, want empty", def.Logging.Level)
	}

	// Explicit passthrough.
	internal := Config{EnableHistory: true, LogLevel: "debug"}.toInternal()
	if !internal.Consent.History.Enabled {
		t.Errorf("History.Enabled = false, want true")
	}
	if internal.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want debug", internal.Logging.Level)
	}
}
