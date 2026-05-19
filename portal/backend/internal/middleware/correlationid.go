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

// Package middleware contains HTTP middleware helpers used by the BFF.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

const correlationHeader = "X-Correlation-ID"
const maxCorrelationIDLength = 64

var randomRead = rand.Read
var fallbackSequence uint64

// CorrelationID ensures each request has a correlation ID and mirrors it in responses.
func CorrelationID(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(correlationHeader)
		if !isValidCorrelationID(id) {
			id = newCorrelationID()
		}
		r.Header.Set(correlationHeader, id)
		w.Header().Set(correlationHeader, id)

		log.Debug("request received", "method", r.Method, "path", r.URL.Path, "correlation_id", id)
		next.ServeHTTP(w, r)
	})
}

func isValidCorrelationID(id string) bool {
	if id == "" || len(id) > maxCorrelationIDLength {
		return false
	}

	for _, r := range id {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') {
			continue
		}

		switch r {
		case '-', '_', '.', ':':
			continue
		default:
			return false
		}
	}

	return true
}

func newCorrelationID() string {
	buf := make([]byte, 16)
	if _, err := randomRead(buf); err != nil {
		return fallbackCorrelationID()
	}
	return hex.EncodeToString(buf)
}

func fallbackCorrelationID() string {
	sequence := atomic.AddUint64(&fallbackSequence, 1)
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	pid := strconv.Itoa(os.Getpid())
	seq := strconv.FormatUint(sequence, 36)

	return "fb-" + timestamp + "-" + pid + "-" + seq
}
