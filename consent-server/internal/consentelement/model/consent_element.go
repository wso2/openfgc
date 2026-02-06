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

// Package model provides data models for consent elements.
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/wso2/openfgc/internal/consentelement/validators"
)

// JSONValue represents a JSON value that can be stored in the database
type JSONValue json.RawMessage

// Scan implements the sql.Scanner interface for JSONValue
func (jsonValue *JSONValue) Scan(value interface{}) error {
	if value == nil {
		*jsonValue = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSON value: %v", value)
	}
	*jsonValue = JSONValue(bytes)
	return nil
}

// Value implements the driver.Valuer interface for JSONValue
func (jsonValue JSONValue) Value() (driver.Value, error) {
	if len(jsonValue) == 0 {
		return nil, nil
	}
	return []byte(jsonValue), nil
}

// MarshalJSON implements the json.Marshaler interface
func (jsonValue JSONValue) MarshalJSON() ([]byte, error) {
	if len(jsonValue) == 0 {
		return []byte("null"), nil
	}
	return jsonValue, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (jsonValue *JSONValue) UnmarshalJSON(data []byte) error {
	if jsonValue == nil {
		return fmt.Errorf("JSONValue: UnmarshalJSON on nil pointer")
	}
	*jsonValue = append((*jsonValue)[0:0], data...)
	return nil
}

// ConsentElement represents a consent element entity
type ConsentElement struct {
	ID          string            `json:"id" db:"ID"`
	Name        string            `json:"name" db:"NAME"`
	Description *string           `json:"description,omitempty" db:"DESCRIPTION"`
	Type        string            `json:"type" db:"TYPE"`
	Properties  map[string]string `json:"properties,omitempty" db:"-"`
	OrgID       string            `json:"orgId" db:"ORG_ID"`
}

// ConsentElementMapping represents the CONSENT_PURPOSE_ELEMENT_MAPPING table
type ConsentElementMapping struct {
	ConsentID      string      `db:"CONSENT_ID" json:"consentId"`
	OrgID          string      `db:"ORG_ID" json:"orgId"`
	ElementID      string      `db:"ELEMENT_ID" json:"elementId"`
	Value          interface{} `db:"VALUE" json:"value,omitempty"`
	IsUserApproved bool        `db:"IS_USER_APPROVED" json:"isUserApproved"`
	IsMandatory    bool        `db:"IS_MANDATORY" json:"isMandatory"`
	Name           string      `db:"-" json:"name"` // Element name for convenience (not in mapping table)
}

// ConsentElementCreateRequest represents the request to create a consent element
type ConsentElementCreateRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type" binding:"required"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// ConsentElementUpdateRequest represents the request to update a consent element
// All fields are required - no partial updates allowed
type ConsentElementUpdateRequest struct {
	Name        string            `json:"name" binding:"required,max=255"`
	Description *string           `json:"description,omitempty" binding:"omitempty,max=1024"`
	Type        string            `json:"type" binding:"required"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// ConsentElementResponse represents the response for consent element operations
type ConsentElementResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Type        string            `json:"type"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// ListResponse represents a list of consent elements
type ListResponse struct {
	Elements []ConsentElementResponse `json:"elements"`
	Total    int                      `json:"total"`
}

// ToConsentElementResponse converts ConsentElement to ConsentElementResponse
func (cp *ConsentElement) ToConsentElementResponse() *ConsentElementResponse {
	return &ConsentElementResponse{
		ID:          cp.ID,
		Name:        cp.Name,
		Description: cp.Description,
		Type:        cp.Type,
		Properties:  cp.Properties,
	}
}

// ValidateElementType validates that the element type is registered in the handler registry
func ValidateElementType(typeVal string) error {
	_, err := validators.GetHandler(typeVal)
	if err != nil {
		// Get all registered types for helpful error message
		registeredTypes := validators.GetAllHandlerTypes()
		return fmt.Errorf("invalid element type '%s': must be one of %v", typeVal, registeredTypes)
	}
	return nil
}

// ConsentElementProperty represents properties for consent elements
type ConsentElementProperty struct {
	ElementID string `db:"ELEMENT_ID"`
	Key       string `db:"ATT_KEY"`
	Value     string `db:"ATT_VALUE"`
	OrgID     string `db:"ORG_ID"`
}
