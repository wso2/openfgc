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

// ConsentStatusAudit represents the CONSENT_STATUS_AUDIT table
type ConsentStatusAudit struct {
	StatusAuditID  string  `db:"STATUS_AUDIT_ID" json:"statusAuditId"`
	ConsentID      string  `db:"CONSENT_ID" json:"consentId"`
	CurrentStatus  string  `db:"CURRENT_STATUS" json:"currentStatus"`
	ActionTime     int64   `db:"ACTION_TIME" json:"actionTime"`
	Reason         *string `db:"REASON" json:"reason,omitempty"`
	ActionBy       *string `db:"ACTION_BY" json:"actionBy,omitempty"`
	PreviousStatus *string `db:"PREVIOUS_STATUS" json:"previousStatus,omitempty"`
	OrgID          string  `db:"ORG_ID" json:"orgId"`
}

// StatusAudit is an alias for ConsentStatusAudit for backward compatibility
type StatusAudit = ConsentStatusAudit

// ConsentStatusAuditCreateRequest represents the request for creating a status audit entry
type ConsentStatusAuditCreateRequest struct {
	ConsentID      string  `json:"consentId" binding:"required"`
	CurrentStatus  string  `json:"currentStatus" binding:"required"`
	Reason         *string `json:"reason,omitempty"`
	ActionBy       *string `json:"actionBy,omitempty"`
	PreviousStatus *string `json:"previousStatus,omitempty"`
}

// ConsentStatusAuditResponse represents the response for status audit operations
type ConsentStatusAuditResponse struct {
	StatusAuditID  string  `json:"statusAuditId"`
	ConsentID      string  `json:"consentId"`
	CurrentStatus  string  `json:"currentStatus"`
	ActionTime     int64   `json:"actionTime"`
	Reason         *string `json:"reason,omitempty"`
	ActionBy       *string `json:"actionBy,omitempty"`
	PreviousStatus *string `json:"previousStatus,omitempty"`
	OrgID          string  `json:"orgId"`
}

// ConsentStatusAuditListResponse represents the list of audit entries
type ConsentStatusAuditListResponse struct {
	Data []ConsentStatusAuditResponse `json:"data"`
}
