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

package model

// ConsentAttribute represents the CONSENT_ATTRIBUTE table
type ConsentAttribute struct {
	ConsentID string `db:"CONSENT_ID" json:"consentId"`
	AttKey    string `db:"ATT_KEY" json:"key"`
	AttValue  string `db:"ATT_VALUE" json:"value"`
	OrgID     string `db:"ORG_ID" json:"orgId"`
}

// ConsentAttributeCreateRequest represents the request for creating consent attributes
type ConsentAttributeCreateRequest struct {
	ConsentID  string            `json:"consentId" binding:"required"`
	Attributes map[string]string `json:"attributes" binding:"required"`
}

// ConsentAttributeUpdateRequest represents the request for updating consent attributes
type ConsentAttributeUpdateRequest struct {
	Attributes map[string]string `json:"attributes" binding:"required"`
}

// ConsentAttributeResponse represents the response for attribute operations
type ConsentAttributeResponse struct {
	ConsentID  string            `json:"consentId"`
	Attributes map[string]string `json:"attributes"`
	OrgID      string            `json:"orgId"`
}

// ConsentAttributeSearchResponse represents the response for attribute search
type ConsentAttributeSearchResponse struct {
	ConsentIDs []string `json:"consentIds"`
	Count      int      `json:"count"`
}
