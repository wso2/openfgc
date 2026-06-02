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

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPointersEqual(t *testing.T) {
	first := 10
	same := 10
	different := 20

	require.True(t, PointersEqual[int](nil, nil))
	require.False(t, PointersEqual(&first, nil))
	require.False(t, PointersEqual(nil, &same))
	require.True(t, PointersEqual(&first, &same))
	require.False(t, PointersEqual(&first, &different))
}

func TestCanonicalJSONString(t *testing.T) {
	first := CanonicalJSONString(`{"region":"EU","department":"customer_service"}`)
	second := CanonicalJSONString(`{"department":"customer_service","region":"EU"}`)

	require.Equal(t, first, second)
	require.Equal(t, "not-json", CanonicalJSONString("not-json"))
	require.Empty(t, CanonicalJSONString(""))
}

func TestCanonicalJSONValue(t *testing.T) {
	value := map[string]interface{}{
		"department": "customer_service",
		"regions":    []interface{}{"EU", "LK"},
	}

	require.Equal(t, `{"department":"customer_service","regions":["EU","LK"]}`, CanonicalJSONValue(value))
	require.Empty(t, CanonicalJSONValue(nil))
}
