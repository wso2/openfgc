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

package consentelement

// Request models - API expects an array of ConsentElementCreateRequest
type ConsentElementCreateRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// Update request model - PUT /consent-elements/{id}
type ConsentElementUpdateRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// Response models
type ElementResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Type        string            `json:"type"`
	Properties  map[string]string `json:"properties,omitempty"`
	CreatedAt   string            `json:"createdAt,omitempty"`
	UpdatedAt   string            `json:"updatedAt,omitempty"`
}

type ElementListResponse struct {
	Data     []ElementResponse `json:"data"`
	Metadata Metadata          `json:"metadata"`
}

type Metadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
	Limit  int `json:"limit"`
}

type ElementCreateResponse struct {
	Data    []ElementResponse `json:"data"`
	Message string            `json:"message"`
}

type ErrorResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	TraceID     string `json:"traceId,omitempty"`
}
