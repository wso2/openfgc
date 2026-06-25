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

import "encoding/json"

// ConsentHistory represents the CONSENT_HISTORY table.
type ConsentHistory struct {
	HistoryID  string          `db:"HISTORY_ID"`
	ConsentID  string          `db:"CONSENT_ID"`
	OrgID      string          `db:"ORG_ID"`
	ActionTime int64           `db:"ACTION_TIME"`
	ActionBy   *string         `db:"ACTION_BY"`
	Reason     *string         `db:"REASON"`
	Snapshot   json.RawMessage `db:"SNAPSHOT"`
}

// ConsentHistoryOutput is the service-layer representation of one history entry.
type ConsentHistoryOutput struct {
	HistoryID  string
	ConsentID  string
	OrgID      string
	ActionTime int64
	ActionBy   *string
	Reason     *string
	Snapshot   json.RawMessage
}

// ConsentHistoryListOutput is the service-layer representation of consent history.
type ConsentHistoryListOutput struct {
	ID      string
	History []ConsentHistoryOutput
}

// ConsentHistoryResponse represents a single history response item.
type ConsentHistoryResponse struct {
	HistoryID  string          `json:"historyId"`
	ActionTime int64           `json:"actionTime"`
	ActionBy   *string         `json:"actionBy,omitempty"`
	Reason     *string         `json:"reason,omitempty"`
	Snapshot   json.RawMessage `json:"snapshot,omitempty"`
}

// ConsentHistoryListResponse represents consent history.
type ConsentHistoryListResponse struct {
	ID      string                   `json:"id"`
	History []ConsentHistoryResponse `json:"history"`
}
