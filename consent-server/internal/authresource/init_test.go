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

package authresource

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/system/stores"
)

func TestInitialize(t *testing.T) {
	mux := http.NewServeMux()
	registry := &stores.StoreRegistry{}

	service := Initialize(mux, registry)

	require.NotNil(t, service)
}

func TestNewAuthResourceHandler(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	require.NotNil(t, handler)
	require.NotNil(t, handler.service)
}

func TestNewAuthResourceService(t *testing.T) {
	registry := &stores.StoreRegistry{}
	service := newAuthResourceService(registry)

	require.NotNil(t, service)
}

func TestRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	// Should not panic
	registerRoutes(mux, handler)
}
