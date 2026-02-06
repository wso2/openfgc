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

package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/consent/model"
)

func TestValidateConsentCreateRequest_Success(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type: "accounts",
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "org-1")
	require.NoError(t, err)
}

func TestValidateConsentCreateRequest_MissingType(t *testing.T) {
	req := model.ConsentAPIRequest{
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "type is required")
}

func TestValidateConsentCreateRequest_TypeTooLong(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type: string(make([]byte, 65)),
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "type must be at most 64 characters")
}

func TestValidateConsentGetRequest_Success(t *testing.T) {
	err := ValidateConsentGetRequest("consent-123", "org-1")
	require.NoError(t, err)
}

func TestValidateConsentGetRequest_EmptyConsentID(t *testing.T) {
	err := ValidateConsentGetRequest("", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "consent ID cannot be empty")
}

func TestValidateConsentGetRequest_EmptyOrgID(t *testing.T) {
	err := ValidateConsentGetRequest("consent-123", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization ID cannot be empty")
}

func TestValidateConsentGetRequest_ConsentIDTooLong(t *testing.T) {
	err := ValidateConsentGetRequest(string(make([]byte, 256)), "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "consent ID too long")
}

func TestValidateConsentGetRequest_OrgIDTooLong(t *testing.T) {
	err := ValidateConsentGetRequest("consent-123", string(make([]byte, 256)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization ID too long")
}

func TestValidateConsentCreateRequest_MissingClientID(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type:           "accounts",
		Authorizations: []model.AuthorizationAPIRequest{{Type: "accounts"}},
	}
	err := ValidateConsentCreateRequest(req, "", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "clientID is required")
}

func TestValidateConsentCreateRequest_MissingOrgID(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type:           "accounts",
		Authorizations: []model.AuthorizationAPIRequest{{Type: "accounts"}},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "orgID is required")
}

func TestValidateConsentCreateRequest_MissingAuthType(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type:           "accounts",
		Authorizations: []model.AuthorizationAPIRequest{{Type: ""}},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorizations[0].type is required")
}

func TestValidateConsentCreateRequest_NegativeValidityTime(t *testing.T) {
	negativeTime := int64(-100)
	req := model.ConsentAPIRequest{
		Type:           "accounts",
		ValidityTime:   &negativeTime,
		Authorizations: []model.AuthorizationAPIRequest{{Type: "accounts"}},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "validityTime must be non-negative")
}

func TestValidateConsentCreateRequest_NegativeFrequency(t *testing.T) {
	negativeFreq := -5
	req := model.ConsentAPIRequest{
		Type:           "accounts",
		Frequency:      &negativeFreq,
		Authorizations: []model.AuthorizationAPIRequest{{Type: "accounts"}},
	}
	err := ValidateConsentCreateRequest(req, "client-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "frequency must be non-negative")
}

func TestValidateConsentUpdateRequest_Success(t *testing.T) {
	req := model.ConsentAPIUpdateRequest{Type: "payments"}
	err := ValidateConsentUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateConsentUpdateRequest_NoFieldsProvided(t *testing.T) {
	req := model.ConsentAPIUpdateRequest{}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one field must be provided")
}

func TestValidateConsentUpdateRequest_TypeTooLong(t *testing.T) {
	req := model.ConsentAPIUpdateRequest{Type: string(make([]byte, 65))}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "type must be at most 64 characters")
}

func TestValidateConsentUpdateRequest_NegativeValidityTime(t *testing.T) {
	negativeTime := int64(-100)
	req := model.ConsentAPIUpdateRequest{ValidityTime: &negativeTime}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validityTime must be non-negative")
}

func TestValidateConsentUpdateRequest_NegativeFrequency(t *testing.T) {
	negativeFreq := -5
	req := model.ConsentAPIUpdateRequest{Frequency: &negativeFreq}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "frequency must be non-negative")
}

func TestValidateConsentUpdateRequest_MissingAuthType(t *testing.T) {
	auths := []model.AuthorizationAPIRequest{{Type: ""}}
	req := model.ConsentAPIUpdateRequest{Authorizations: auths}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorizations[0].type is required")
}
