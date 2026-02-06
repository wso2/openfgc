/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
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

package utils

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UUIDUtilTestSuite struct {
	suite.Suite
}

func TestUUIDUtilSuite(t *testing.T) {
	suite.Run(t, new(UUIDUtilTestSuite))
}

func (suite *UUIDUtilTestSuite) TestGenerateUUID() {
	uuid := GenerateUUID()

	// RFC 4122/9562 compliant UUID format: 8-4-4-4-12 hexadecimal characters
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	assert.True(suite.T(), uuidPattern.MatchString(uuid), "UUID should match the RFC 9562 format")

	// The 13th character is the first character of the 3rd group and should be '4' for version 4 UUIDs
	assert.Equal(suite.T(), "4", string(uuid[14]), "UUID version should be 4")

	// The 17th character is the first character of the 4th group
	// The first 2-3 bits should be '10' for variant 1 UUIDs
	// For a hex representation, this means the first character should be 8, 9, A, or B
	variantChar := uuid[19]
	assert.Contains(suite.T(), "89ab", string(variantChar), "UUID variant should be 10xx (RFC 4122/9562)")
}

func (suite *UUIDUtilTestSuite) TestGenerateUUIDUniqueness() {
	uuids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		uuid := GenerateUUID()
		_, exists := uuids[uuid]
		assert.False(suite.T(), exists, "Generated UUIDs should be unique")
		uuids[uuid] = true
	}

	assert.Equal(suite.T(), 100, len(uuids))
}

func (suite *UUIDUtilTestSuite) TestGenerateUUIDLength() {
	uuid := GenerateUUID()

	// UUID string format should be exactly 36 characters (32 hex digits + 4 hyphens)
	assert.Equal(suite.T(), 36, len(uuid), "UUID should be 36 characters long")
}

func (suite *UUIDUtilTestSuite) TestIsValidUUID() {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "ValidLowercaseUUID",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "ValidUppercaseUUID",
			input:    "550E8400-E29B-41D4-A716-446655440000",
			expected: true,
		},
		{
			name:     "UUIDWithWhitespace",
			input:    " 550e8400-e29b-41d4-a716-446655440000 ",
			expected: false,
		},
		{
			name:     "InvalidCharacters",
			input:    "550e8400-e29b-41d4-a716-44665544000Z",
			expected: false,
		},
		{
			name:     "InvalidLength",
			input:    "550e8400-e29b-41d4-a716-44665544",
			expected: false,
		},
		{
			name:     "EmptyString",
			input:    "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := IsValidUUID(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func (suite *UUIDUtilTestSuite) TestGenerateUUIDv7() {
	uuid, err := GenerateUUIDv7()

	assert.NoError(suite.T(), err, "GenerateUUIDv7 should not return error for valid system time")
	assert.NotEmpty(suite.T(), uuid, "Generated UUIDv7 should not be empty")

	// RFC 9562 compliant UUID format: 8-4-4-4-12 hexadecimal characters
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	assert.True(suite.T(), uuidPattern.MatchString(uuid), "UUIDv7 should match the RFC 9562 format")

	// The 13th character is the first character of the 3rd group and should be '7' for version 7 UUIDs
	assert.Equal(suite.T(), "7", string(uuid[14]), "UUID version should be 7")

	// The 17th character is the first character of the 4th group
	// The first 2-3 bits should be '10' for variant 1 UUIDs
	// For a hex representation, this means the first character should be 8, 9, A, or B
	variantChar := uuid[19]
	assert.Contains(suite.T(), "89ab", string(variantChar), "UUID variant should be 10xx (RFC 9562)")
}

func (suite *UUIDUtilTestSuite) TestGenerateUUIDv7Uniqueness() {
	uuids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		uuid, err := GenerateUUIDv7()
		assert.NoError(suite.T(), err)
		_, exists := uuids[uuid]
		assert.False(suite.T(), exists, "Generated UUIDv7s should be unique")
		uuids[uuid] = true
	}

	assert.Equal(suite.T(), 100, len(uuids))
}

func (suite *UUIDUtilTestSuite) TestGenerateUUIDv7Length() {
	uuid, err := GenerateUUIDv7()

	assert.NoError(suite.T(), err)
	// UUID string format should be exactly 36 characters (32 hex digits + 4 hyphens)
	assert.Equal(suite.T(), 36, len(uuid), "UUIDv7 should be 36 characters long")
}

func (suite *UUIDUtilTestSuite) TestGenerateUUIDv7TimeOrdered() {
	// Generate multiple UUIDs with small delays and verify they are in increasing order
	uuid1, err1 := GenerateUUIDv7()
	assert.NoError(suite.T(), err1)

	// Small delay to ensure different timestamps
	time.Sleep(2 * time.Millisecond)

	uuid2, err2 := GenerateUUIDv7()
	assert.NoError(suite.T(), err2)

	time.Sleep(2 * time.Millisecond)

	uuid3, err3 := GenerateUUIDv7()
	assert.NoError(suite.T(), err3)

	// UUIDv7 should be lexicographically sortable due to time-ordered prefix
	assert.True(suite.T(), uuid1 < uuid2, "UUIDv7 should be time-ordered")
	assert.True(suite.T(), uuid2 < uuid3, "UUIDv7 should be time-ordered")
}
