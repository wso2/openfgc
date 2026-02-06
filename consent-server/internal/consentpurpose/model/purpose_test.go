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

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func stringPtr(s string) *string {
	return &s
}

func TestToResponse(t *testing.T) {
	testCases := []struct {
		name     string
		purpose  ConsentPurpose
		expected Response
	}{
		{
			name: "Complete purpose with all fields",
			purpose: ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: stringPtr("Test Description"),
				ClientID:    "client-456",
				Elements: []PurposeElement{
					{
						ElementID:   "elem-1",
						ElementName: "Element 1",
						IsMandatory: true,
					},
					{
						ElementID:   "elem-2",
						ElementName: "Element 2",
						IsMandatory: false,
					},
				},
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
				OrgID:       "org-789",
			},
			expected: Response{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: stringPtr("Test Description"),
				ClientID:    "client-456",
				Elements: []PurposeElement{
					{
						ElementID:   "elem-1",
						ElementName: "Element 1",
						IsMandatory: true,
					},
					{
						ElementID:   "elem-2",
						ElementName: "Element 2",
						IsMandatory: false,
					},
				},
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
			},
		},
		{
			name: "Purpose without description",
			purpose: ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: nil,
				ClientID:    "client-456",
				Elements: []PurposeElement{
					{
						ElementID:   "elem-1",
						ElementName: "Element 1",
						IsMandatory: true,
					},
				},
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
				OrgID:       "org-789",
			},
			expected: Response{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: nil,
				ClientID:    "client-456",
				Elements: []PurposeElement{
					{
						ElementID:   "elem-1",
						ElementName: "Element 1",
						IsMandatory: true,
					},
				},
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
			},
		},
		{
			name: "Purpose with no elements",
			purpose: ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: stringPtr("Test Description"),
				ClientID:    "client-456",
				Elements:    []PurposeElement{},
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
				OrgID:       "org-789",
			},
			expected: Response{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: stringPtr("Test Description"),
				ClientID:    "client-456",
				Elements:    []PurposeElement{},
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
			},
		},
		{
			name: "Purpose with nil elements slice",
			purpose: ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: stringPtr("Test Description"),
				ClientID:    "client-456",
				Elements:    nil,
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
				OrgID:       "org-789",
			},
			expected: Response{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: stringPtr("Test Description"),
				ClientID:    "client-456",
				Elements:    nil,
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
			},
		},
		{
			name: "Purpose with minimal fields",
			purpose: ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				ClientID:    "client-456",
				OrgID:       "org-789",
				CreatedTime: 0,
				UpdatedTime: 0,
			},
			expected: Response{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				ClientID:    "client-456",
				CreatedTime: 0,
				UpdatedTime: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.purpose.ToResponse()

			require.Equal(t, tc.expected.ID, result.ID)
			require.Equal(t, tc.expected.Name, result.Name)
			require.Equal(t, tc.expected.ClientID, result.ClientID)
			require.Equal(t, tc.expected.CreatedTime, result.CreatedTime)
			require.Equal(t, tc.expected.UpdatedTime, result.UpdatedTime)

			if tc.expected.Description == nil {
				require.Nil(t, result.Description)
			} else {
				require.NotNil(t, result.Description)
				require.Equal(t, *tc.expected.Description, *result.Description)
			}

			require.Equal(t, len(tc.expected.Elements), len(result.Elements))
			for i, expectedElem := range tc.expected.Elements {
				require.Equal(t, expectedElem.ElementID, result.Elements[i].ElementID)
				require.Equal(t, expectedElem.ElementName, result.Elements[i].ElementName)
				require.Equal(t, expectedElem.IsMandatory, result.Elements[i].IsMandatory)
			}
		})
	}
}
