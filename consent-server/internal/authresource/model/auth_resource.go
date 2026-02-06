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

// ConsentAuthResource represents the CONSENT_AUTH_RESOURCE table
type ConsentAuthResource struct {
	AuthID      string      `db:"AUTH_ID" json:"authId"`
	ConsentID   string      `db:"CONSENT_ID" json:"consentId"`
	AuthType    string      `db:"AUTH_TYPE" json:"authType"`
	UserID      *string     `db:"USER_ID" json:"userId,omitempty"`
	AuthStatus  string      `db:"AUTH_STATUS" json:"authStatus"`
	UpdatedTime int64       `db:"UPDATED_TIME" json:"updatedTime"`
	Resources   *string     `db:"RESOURCES" json:"-"`
	ResourceObj interface{} `db:"-" json:"resources,omitempty"`
	OrgID       string      `db:"ORG_ID" json:"orgId"`
}

// ConsentAuthResourceCreateRequest represents the request payload for creating an authorization resource
type ConsentAuthResourceCreateRequest struct {
	AuthType   string      `json:"type" binding:"required"`
	UserID     *string     `json:"userId,omitempty"`
	AuthStatus string      `json:"status" binding:"required"`
	Resources  interface{} `json:"resources,omitempty"`
}

// ConsentAuthResourceUpdateRequest represents the request payload for updating an authorization resource
type ConsentAuthResourceUpdateRequest struct {
	AuthStatus string      `json:"status,omitempty"`
	UserID     *string     `json:"userId,omitempty"`
	Resources  interface{} `json:"resources,omitempty"`
}

// ConsentAuthResourceResponse represents the response for authorization resource operations
type ConsentAuthResourceResponse struct {
	AuthID      string      `json:"id"`
	AuthType    string      `json:"type"`
	UserID      *string     `json:"userId,omitempty"`
	AuthStatus  string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ConsentAuthResourceListResponse represents the response for listing authorization resources
type ConsentAuthResourceListResponse struct {
	Data []ConsentAuthResourceResponse `json:"data"`
}

// Type aliases for backward compatibility with service layer
type AuthResource = ConsentAuthResource
type CreateRequest = ConsentAuthResourceCreateRequest
type UpdateRequest = ConsentAuthResourceUpdateRequest
type Response = ConsentAuthResourceResponse
type ListResponse = ConsentAuthResourceListResponse
