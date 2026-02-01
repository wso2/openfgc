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

package transaction

import (
	"context"
	"database/sql"
	"fmt"
)

// DBInterface defines the minimal interface needed for transaction management.
type DBInterface interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Transactioner provides automatic transaction lifecycle management.
// It handles transaction begin, commit, rollback, and panic recovery.
// It also supports nested transactions by detecting and reusing existing transactions from context.
type Transactioner interface {
	// Transact executes the given function within a transaction.
	// If a transaction already exists in the context, it reuses it (nested transaction support).
	// Otherwise, it begins a new transaction, executes the function, and commits or rolls back
	// based on the result. Panics are automatically recovered and the transaction is rolled back.
	Transact(ctx context.Context, txFunc func(context.Context) error) error
}

// transactioner is the implementation of Transactioner.
type transactioner struct {
	db DBInterface
}

// NewTransactioner creates a new Transactioner instance.
func NewTransactioner(db DBInterface) Transactioner {
	return &transactioner{db: db}
}

// Transact executes the given function within a transaction.
// It provides automatic transaction lifecycle management:
// - Detects existing transaction in context (nested transaction support)
// - Begins new transaction if needed
// - Commits on success
// - Rolls back on error
// - Rolls back and captures panic as error
func (t *transactioner) Transact(ctx context.Context, txFunc func(context.Context) error) (err error) {
	// Check if we're already in a transaction (nested transaction support)
	if HasTx(ctx) {
		// Reuse existing transaction - don't begin new one
		return txFunc(ctx)
	}

	// Begin new transaction
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Setup deferred commit/rollback with panic recovery
	defer func() {
		if p := recover(); p != nil {
			// Panic occurred - rollback and convert to error
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("transaction aborted due to panic: %v (rollback error: %v)", p, rbErr)
			} else {
				err = fmt.Errorf("transaction aborted due to panic: %v", p)
			}
		} else if err != nil {
			// Error occurred - rollback
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("transaction error: %w (rollback error: %v)", err, rbErr)
			}
		} else {
			// Success - commit
			if commitErr := tx.Commit(); commitErr != nil {
				err = fmt.Errorf("failed to commit transaction: %w", commitErr)
			}
		}
	}()

	// Create context with transaction and execute function
	txCtx := WithTx(ctx, tx)
	err = txFunc(txCtx)
	return err
}
