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

package middleware

import (
	"errors"
	"testing"
)

func TestIsValidCorrelationID_RejectsNonASCIICharacters(t *testing.T) {
	if isValidCorrelationID("request-１２３") {
		t.Fatal("expected full-width digits to be rejected")
	}
	if isValidCorrelationID("request-αβγ") {
		t.Fatal("expected non-ASCII letters to be rejected")
	}
}

func TestNewCorrelationID_UsesUniqueFallbackWhenRandomFails(t *testing.T) {
	originalRandomRead := randomRead
	randomRead = func(_ []byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}
	t.Cleanup(func() {
		randomRead = originalRandomRead
	})

	first := newCorrelationID()
	second := newCorrelationID()

	if first == second {
		t.Fatalf("expected unique fallback ids, got same value %q", first)
	}
	if !isValidCorrelationID(first) {
		t.Fatalf("expected first fallback id to be valid, got %q", first)
	}
	if !isValidCorrelationID(second) {
		t.Fatalf("expected second fallback id to be valid, got %q", second)
	}
}
