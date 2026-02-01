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

package consentpurpose

// PurposeCreateRequest represents the request payload for creating a purpose
type PurposeCreateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Elements    []PurposeElement `json:"elements"`
}

// PurposeUpdateRequest represents the request payload for updating a purpose
type PurposeUpdateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Elements    []PurposeElement `json:"elements"`
}

// PurposeElement represents an element within a purpose
type PurposeElement struct {
	Name        string `json:"name"`
	IsMandatory bool   `json:"isMandatory"`
}

// PurposeResponse represents the response for a purpose
type PurposeResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	ClientID    string           `json:"clientId"`
	Elements    []PurposeElement `json:"elements"`
	CreatedTime int64            `json:"createdTime"`
	UpdatedTime int64            `json:"updatedTime"`
}

// PurposeListResponse represents the response for listing purposes
type PurposeListResponse struct {
	Data     []PurposeResponse   `json:"data"`
	Metadata PurposeListMetadata `json:"metadata"`
}

// PurposeListMetadata represents metadata for list operations
type PurposeListMetadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ErrorResponse represents error response from the API
type ErrorResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
	TraceID     string `json:"traceId"`
}
