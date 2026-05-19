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
	"net"
	"net/http"
	"net/url"
	"strings"
)

// CORSOptions defines configurable CORS policy for browser clients.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// CORS applies origin checks and preflight handling for allowed browser origins.
func CORS(next http.Handler, options CORSOptions) http.Handler {
	allowedOrigins := toSet(options.AllowedOrigins)
	allowMethods := strings.Join(options.AllowedMethods, ", ")
	allowHeaders := strings.Join(options.AllowedHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if isSameOrigin(origin, r) {
			next.ServeHTTP(w, r)
			return
		}

		if _, ok := allowedOrigins[origin]; !ok {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		appendVary(w.Header(), "Origin")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		if options.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSameOrigin(origin string, r *http.Request) bool {
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return false
	}

	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	if !strings.EqualFold(parsedOrigin.Scheme, requestScheme) {
		return false
	}

	originHost, originPort := splitHostPort(parsedOrigin.Host)
	requestHost, requestPort := splitHostPort(r.Host)
	if !strings.EqualFold(originHost, requestHost) {
		return false
	}

	if originPort == "" {
		originPort = defaultPort(requestScheme)
	}
	if requestPort == "" {
		requestPort = defaultPort(requestScheme)
	}
	return originPort == requestPort
}

func splitHostPort(hostPort string) (string, string) {
	host := strings.TrimSpace(hostPort)
	if host == "" {
		return "", ""
	}

	if parsedHost := (&url.URL{Host: host}).Hostname(); parsedHost != "" {
		return parsedHost, (&url.URL{Host: host}).Port()
	}

	hostOnly, port, err := net.SplitHostPort(host)
	if err == nil {
		return hostOnly, port
	}
	return host, ""
}

func defaultPort(scheme string) string {
	switch strings.ToLower(scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}

func appendVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}
