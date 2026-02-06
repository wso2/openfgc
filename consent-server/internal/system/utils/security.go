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

package utils

import (
	"html"
	"net/url"
	"strings"
	"unicode"
)

func SanitizeString(input string) string {
	trimmed := strings.TrimSpace(input)
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, trimmed)
	return html.EscapeString(cleaned)
}

// IsValidURI validates a URI with a safe allowlist of schemes (http, https).
// Returns true only if the URI parses successfully, has a scheme in the allowlist,
// and has a non-empty host.
func IsValidURI(uri string) bool {
	return IsValidURIWithSchemes(uri, []string{"http", "https"})
}

// IsValidURIWithSchemes validates a URI against an explicit list of allowed schemes.
// Returns true only if the URI parses successfully, has a scheme in the allowlist,
// and has a non-empty host for network URIs.
func IsValidURIWithSchemes(uri string, allowedSchemes []string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// Require non-empty scheme and host
	if parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	// Check if scheme is in the allowlist
	for _, allowed := range allowedSchemes {
		if strings.EqualFold(parsed.Scheme, allowed) {
			return true
		}
	}

	return false
}
