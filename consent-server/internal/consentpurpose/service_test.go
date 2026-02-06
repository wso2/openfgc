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

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/consentpurpose/model"
	"github.com/wso2/openfgc/internal/system/stores"
)

// TestValidateCreateRequest tests validation of create requests
func TestValidateCreateRequest(t *testing.T) {
	testCases := []struct {
		name          string
		request       model.CreateRequest
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid request",
			request: model.CreateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError: false,
		},
		{
			name: "Missing name",
			request: model.CreateRequest{
				Name:        "",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "name is required",
		},
		{
			name: "Name too long",
			request: model.CreateRequest{
				Name:        strings.Repeat("a", 256),
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "name must not exceed 255 characters",
		},
		{
			name: "Description too long",
			request: model.CreateRequest{
				Name:        "Test Purpose",
				Description: strings.Repeat("a", 1025),
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "description must not exceed 1024 characters",
		},
		{
			name: "No elements",
			request: model.CreateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements:    []model.ElementInput{},
			},
			expectError:   true,
			errorContains: "at least one element is required",
		},
		{
			name: "Empty element name",
			request: model.CreateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "element names cannot be empty",
		},
		{
			name: "Multiple elements",
			request: model.CreateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
					{ElementName: "element2", IsMandatory: false},
				},
			},
			expectError: false,
		},
		{
			name: "Max length name valid",
			request: model.CreateRequest{
				Name:        strings.Repeat("a", 255),
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError: false,
		},
		{
			name: "Max length description valid",
			request: model.CreateRequest{
				Name:        "Test Purpose",
				Description: strings.Repeat("a", 1024),
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError: false,
		},
	}

	registry := &stores.StoreRegistry{}
	service := &consentPurposeService{stores: registry}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.validateCreateRequest(tc.request)

			if tc.expectError {
				require.NotNil(t, err)
				require.Contains(t, err.Description, tc.errorContains)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

// TestValidateUpdateRequest tests validation of update requests
func TestValidateUpdateRequest(t *testing.T) {
	testCases := []struct {
		name          string
		request       model.UpdateRequest
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid request",
			request: model.UpdateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError: false,
		},
		{
			name: "Missing name",
			request: model.UpdateRequest{
				Name:        "",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "name is required",
		},
		{
			name: "Name too long",
			request: model.UpdateRequest{
				Name:        strings.Repeat("a", 256),
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "name must not exceed 255 characters",
		},
		{
			name: "Description too long",
			request: model.UpdateRequest{
				Name:        "Test Purpose",
				Description: strings.Repeat("a", 1025),
				Elements: []model.ElementInput{
					{ElementName: "element1", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "description must not exceed 1024 characters",
		},
		{
			name: "No elements",
			request: model.UpdateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements:    []model.ElementInput{},
			},
			expectError:   true,
			errorContains: "at least one element is required",
		},
		{
			name: "Empty element name",
			request: model.UpdateRequest{
				Name:        "Test Purpose",
				Description: "Test Description",
				Elements: []model.ElementInput{
					{ElementName: "", IsMandatory: true},
				},
			},
			expectError:   true,
			errorContains: "element names cannot be empty",
		},
	}

	registry := &stores.StoreRegistry{}
	service := &consentPurposeService{stores: registry}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.validateUpdateRequest(tc.request)

			if tc.expectError {
				require.NotNil(t, err)
				require.Contains(t, err.Description, tc.errorContains)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

// TestCheckDuplicateElementNames tests duplicate element detection
func TestCheckDuplicateElementNames(t *testing.T) {
	testCases := []struct {
		name          string
		elements      []model.ElementInput
		expectError   bool
		errorContains string
	}{
		{
			name: "No duplicates",
			elements: []model.ElementInput{
				{ElementName: "element1", IsMandatory: true},
				{ElementName: "element2", IsMandatory: false},
				{ElementName: "element3", IsMandatory: true},
			},
			expectError: false,
		},
		{
			name: "Duplicate element",
			elements: []model.ElementInput{
				{ElementName: "element1", IsMandatory: true},
				{ElementName: "element2", IsMandatory: false},
				{ElementName: "element1", IsMandatory: true},
			},
			expectError:   true,
			errorContains: "duplicate element 'element1'",
		},
		{
			name: "Single element",
			elements: []model.ElementInput{
				{ElementName: "element1", IsMandatory: true},
			},
			expectError: false,
		},
		{
			name: "Multiple duplicates first",
			elements: []model.ElementInput{
				{ElementName: "element1", IsMandatory: true},
				{ElementName: "element1", IsMandatory: false},
			},
			expectError:   true,
			errorContains: "duplicate element 'element1'",
		},
		{
			name: "Case sensitive duplicates not detected",
			elements: []model.ElementInput{
				{ElementName: "element1", IsMandatory: true},
				{ElementName: "Element1", IsMandatory: false},
			},
			expectError: false,
		},
	}

	registry := &stores.StoreRegistry{}
	service := &consentPurposeService{stores: registry}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.checkDuplicateElementNames(tc.elements)

			if tc.expectError {
				require.NotNil(t, err)
				require.Contains(t, err.Description, tc.errorContains)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
