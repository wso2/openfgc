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

package validators

import (
	"fmt"
	"sync"
)

// ElementTypeHandlerRegistry holds all registered type handlers
type ElementTypeHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ElementTypeHandler
}

var (
	// defaultRegistry is the global registry singleton
	defaultRegistry *ElementTypeHandlerRegistry
)

// init registers all built-in handlers at package init time
func init() {
	defaultRegistry = NewElementTypeHandlerRegistry()

	// Register built-in handlers
	if err := defaultRegistry.Register(&BasicElementTypeHandler{}); err != nil {
		panic(fmt.Sprintf("failed to register BasicElementTypeHandler: %v", err))
	}
	if err := defaultRegistry.Register(&JsonPayloadElementTypeHandler{}); err != nil {
		panic(fmt.Sprintf("failed to register JsonPayloadElementTypeHandler: %v", err))
	}
	if err := defaultRegistry.Register(&ResourceFieldElementTypeHandler{}); err != nil {
		panic(fmt.Sprintf("failed to register ResourceFieldElementTypeHandler: %v", err))
	}
}

// NewElementTypeHandlerRegistry creates a new registry instance
func NewElementTypeHandlerRegistry() *ElementTypeHandlerRegistry {
	return &ElementTypeHandlerRegistry{
		handlers: make(map[string]ElementTypeHandler),
	}
}

// Register adds a handler to the registry
// Returns error if a handler for this type is already registered
func (registry *ElementTypeHandlerRegistry) Register(handler ElementTypeHandler) error {
	typeStr := handler.GetType()

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.handlers[typeStr]; exists {
		return fmt.Errorf("handler for type %q already registered", typeStr)
	}
	registry.handlers[typeStr] = handler
	return nil
}

// Get retrieves a handler by type string
// Returns error if no handler is registered for the type
func (registry *ElementTypeHandlerRegistry) Get(typeStr string) (ElementTypeHandler, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	handler, exists := registry.handlers[typeStr]
	if !exists {
		return nil, fmt.Errorf("no handler registered for the element type %q", typeStr)
	}
	return handler, nil
}

// GetAllTypes returns a list of all registered element types
func (registry *ElementTypeHandlerRegistry) GetAllTypes() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	types := make([]string, 0, len(registry.handlers))
	for typeStr := range registry.handlers {
		types = append(types, typeStr)
	}
	return types
}

// Global helper functions

// GetHandler retrieves a handler from the default registry by type
func GetHandler(typeStr string) (ElementTypeHandler, error) {
	return defaultRegistry.Get(typeStr)
}

// GetAllHandlerTypes returns list of all registered types in default registry
func GetAllHandlerTypes() []string {
	return defaultRegistry.GetAllTypes()
}

// GetDefaultRegistry returns the global registry singleton
func GetDefaultRegistry() *ElementTypeHandlerRegistry {
	return defaultRegistry
}
