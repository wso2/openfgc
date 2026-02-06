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

// Package provider provides database client implementations for executing queries and managing transactions.
package provider

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/transaction"
	"github.com/wso2/openfgc/internal/system/log"
)

// DBClientInterface defines the interface for database operations.
type DBClientInterface interface {
	// Query executes a sql query that returns rows, typically a SELECT, and returns the result as a slice of maps.
	Query(query model.DBQuery, args ...interface{}) ([]map[string]interface{}, error)
	// QueryContext executes a sql query with context awareness (checks for transaction in context).
	QueryContext(ctx context.Context, query model.DBQuery, args ...interface{}) ([]map[string]interface{}, error)
	// Execute executes a sql query without returning data in any rows, and returns number of rows affected.
	Execute(query model.DBQuery, args ...interface{}) (int64, error)
	// ExecuteContext executes a sql query with context awareness (checks for transaction in context).
	ExecuteContext(ctx context.Context, query model.DBQuery, args ...interface{}) (int64, error)
	// BeginTx starts a new database transaction.
	BeginTx() (model.TxInterface, error)
	// GetTransactioner returns a Transactioner for automatic transaction management.
	GetTransactioner() (transaction.Transactioner, error)
}

// DBClient is the implementation of DBClientInterface.
type DBClient struct {
	db     model.DBInterface
	dbType string
}

// NewDBClient creates a new instance of DBClient with the provided database connection.
func NewDBClient(db model.DBInterface, dbType string) DBClientInterface {
	return &DBClient{db: db, dbType: dbType}
}

// Query executes a sql query that returns rows, typically a SELECT, and returns the result as a slice of maps.
// This is a convenience method that calls QueryContext with context.Background().
func (client *DBClient) Query(query model.DBQuery, args ...interface{}) ([]map[string]interface{}, error) {
	return client.QueryContext(context.Background(), query, args...)
}

// QueryContext executes a sql query with context awareness.
// If a transaction exists in the context, it will be used automatically.
func (client *DBClient) QueryContext(ctx context.Context, query model.DBQuery, args ...interface{}) ([]map[string]interface{}, error) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBClient"))
	logger.Debug("Executing query", log.String("query_id", query.GetID()))

	// Check if there's a transaction in the context
	var rows *sql.Rows
	var err error
	if tx := transaction.TxFromContext(ctx); tx != nil {
		// Use transaction from context
		sqlQuery := query.GetQuery(client.dbType)
		rows, err = tx.QueryContext(ctx, sqlQuery, args...)
	} else {
		// Use direct connection
		rows, err = client.db.Query(query, args...)
	}

	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBClient"))
			logger.Error("Error closing rows", log.Error(closeErr))
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		row := make([]interface{}, len(columns))
		rowPointers := make([]interface{}, len(columns))
		for i := range row {
			rowPointers[i] = &row[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			return nil, err
		}

		result := map[string]interface{}{}
		for i, col := range columns {
			// Normalize column names to lowercase for consistency.
			result[strings.ToLower(col)] = row[i]
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Execute a sql query without returning data in any rows, and returns number of rows affected.
// This is a convenience method that calls ExecuteContext with context.Background().
func (client *DBClient) Execute(query model.DBQuery, args ...interface{}) (int64, error) {
	return client.ExecuteContext(context.Background(), query, args...)
}

// ExecuteContext executes a sql query with context awareness.
// If a transaction exists in the context, it will be used automatically.
func (client *DBClient) ExecuteContext(ctx context.Context, query model.DBQuery, args ...interface{}) (int64, error) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBClient"))
	logger.Debug("Executing query", log.String("query_id", query.GetID()))

	// Check if there's a transaction in the context
	var res sql.Result
	var err error
	if tx := transaction.TxFromContext(ctx); tx != nil {
		// Use transaction from context
		sqlQuery := query.GetQuery(client.dbType)
		res, err = tx.ExecContext(ctx, sqlQuery, args...)
	} else {
		// Use direct connection
		res, err = client.db.Exec(query, args...)
	}

	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// BeginTx starts a new database transaction.
func (client *DBClient) BeginTx() (model.TxInterface, error) {
	return client.db.BeginTx()
}

// GetTransactioner returns a Transactioner for automatic transaction management.
func (client *DBClient) GetTransactioner() (transaction.Transactioner, error) {
	// Cast to *model.DB to create transaction wrapper
	if db, ok := client.db.(*model.DB); ok {
		return transaction.NewTransactioner(&transactionDBWrapper{db: db}), nil
	}
	return nil, fmt.Errorf("unsupported database type for transactioner")
}
