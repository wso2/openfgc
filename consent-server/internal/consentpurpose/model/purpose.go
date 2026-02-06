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

// ConsentPurpose represents a consent purpose entity
type ConsentPurpose struct {
	ID          string           `json:"id" db:"ID"`
	Name        string           `json:"name" db:"NAME"`
	Description *string          `json:"description,omitempty" db:"DESCRIPTION"`
	ClientID    string           `json:"clientId" db:"CLIENT_ID"`
	Elements    []PurposeElement `json:"elements" db:"-"`
	CreatedTime int64            `json:"createdTime" db:"CREATED_TIME"`
	UpdatedTime int64            `json:"updatedTime" db:"UPDATED_TIME"`
	OrgID       string           `json:"-" db:"ORG_ID"`
}

// PurposeElement represents an element within a purpose
type PurposeElement struct {
	ElementID   string `json:"-" db:"ELEMENT_ID"` // Internal use
	ElementName string `json:"name" db:"ELEMENT_NAME"`
	IsMandatory bool   `json:"isMandatory" db:"IS_MANDATORY"`
}

// CreateRequest represents a request to create a purpose
type CreateRequest struct {
	Name        string         `json:"name" validate:"required,max=255"`
	Description string         `json:"description,omitempty" validate:"max=1024"`
	Elements    []ElementInput `json:"elements" validate:"required,min=1,dive"`
}

// ElementInput represents element input in create/update requests
type ElementInput struct {
	ElementName string `json:"name" validate:"required"`
	IsMandatory bool   `json:"isMandatory"`
}

// UpdateRequest represents a request to update a purpose
type UpdateRequest struct {
	Name        string         `json:"name" validate:"required,max=255"`
	Description string         `json:"description,omitempty" validate:"max=1024"`
	Elements    []ElementInput `json:"elements" validate:"required,min=1,dive"`
}

// Response represents a purpose response
type Response struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	ClientID    string           `json:"clientId"`
	Elements    []PurposeElement `json:"elements"`
	CreatedTime int64            `json:"createdTime"`
	UpdatedTime int64            `json:"updatedTime"`
}

// ListResponse represents a paginated list of purposes
type ListResponse struct {
	Data     []Response         `json:"data"`
	Metadata PaginationMetadata `json:"metadata"`
}

// PaginationMetadata represents pagination information
type PaginationMetadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
	Limit  int `json:"limit"`
}

// ToResponse converts a ConsentPurpose to Response
func (p *ConsentPurpose) ToResponse() Response {
	return Response{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ClientID:    p.ClientID,
		Elements:    p.Elements,
		CreatedTime: p.CreatedTime,
		UpdatedTime: p.UpdatedTime,
	}
}
