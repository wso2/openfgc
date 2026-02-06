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

// Package stores provides implementations for data store interfaces.
package stores

import (
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores/interfaces"
)

// StoreRegistry holds references to all stores in the application
type StoreRegistry struct {
	// Store instances with typed interfaces
	Consent        interfaces.ConsentStore
	AuthResource   interfaces.AuthResourceStore
	ConsentElement interfaces.ConsentElementStore
	ConsentPurpose interfaces.ConsentPurposeStore
}

// NewStoreRegistry creates a new store registry with all initialized stores
func NewStoreRegistry(
	consentStore interfaces.ConsentStore,
	authResourceStore interfaces.AuthResourceStore,
	consentElementStore interfaces.ConsentElementStore,
	consentPurposeStore interfaces.ConsentPurposeStore,
) *StoreRegistry {
	return &StoreRegistry{
		Consent:        consentStore,
		AuthResource:   authResourceStore,
		ConsentElement: consentElementStore,
		ConsentPurpose: consentPurposeStore,
	}
}

// ExecuteTransaction executes multiple store operations in a single transaction.
// It provides automatic transaction management including rollback on error and panic recovery.
func (r *StoreRegistry) ExecuteTransaction(queries []func(tx dbmodel.TxInterface) error) error {
	logger := log.GetLogger()
	logger.Debug("Starting transaction", log.Int("query_count", len(queries)))

	dbClient, err := provider.GetDBProvider().GetConsentDBClient()
	if err != nil {
		logger.Error("Failed to get database client", log.Error(err))
		return err
	}

	tx, err := dbClient.BeginTx()
	if err != nil {
		logger.Error("Failed to begin transaction", log.Error(err))
		return err
	}

	// Ensure transaction is always closed
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			logger.Error("Transaction panicked and was rolled back", log.Any("panic", p))
			panic(p) // Re-throw panic after rollback
		}
	}()

	for i, query := range queries {
		if err := query(tx); err != nil {
			logger.Warn("Transaction query failed, rolling back",
				log.Error(err),
				log.Int("failed_query_index", i),
			)
			if rbErr := tx.Rollback(); rbErr != nil {
				logger.Error("Failed to rollback transaction", log.Error(rbErr))
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", log.Error(err))
		return err
	}

	logger.Debug("Transaction committed successfully", log.Int("query_count", len(queries)))
	return nil
}
