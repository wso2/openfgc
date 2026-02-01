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
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// DBInterface defines the interface for database operations.
type DBInterface interface {
	Query(query DBQuery, args ...any) (*sql.Rows, error)
	Exec(query DBQuery, args ...any) (sql.Result, error)
	Begin() (*sql.Tx, error)
	BeginTx() (TxInterface, error)
	Close() error
}

// TxInterface defines the interface for transaction operations.
type TxInterface interface {
	Exec(query DBQuery, args ...any) (sql.Result, error)
	Query(query DBQuery, args ...any) (*sql.Rows, error)
	Commit() error
	Rollback() error
}

// DB wraps sqlx.DB and implements DBInterface
type DB struct {
	*sqlx.DB
	DBType string
}

// Query executes a query with DBQuery parameter
func (db *DB) Query(query DBQuery, args ...any) (*sql.Rows, error) {
	sqlQuery := query.GetQuery(db.DBType)
	return db.DB.Query(sqlQuery, args...)
}

// Exec executes a query with DBQuery parameter
func (db *DB) Exec(query DBQuery, args ...any) (sql.Result, error) {
	sqlQuery := query.GetQuery(db.DBType)
	return db.DB.Exec(sqlQuery, args...)
}

// Begin starts a new transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}

// BeginTx starts a new transaction and returns TxInterface
func (db *DB) BeginTx() (TxInterface, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, DBType: db.DBType}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// BeginTxContext starts a new transaction with context (for transaction.DBInterface)
func (db *DB) BeginTxContext(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// Tx wraps sql.Tx and implements TxInterface
type Tx struct {
	*sql.Tx
	DBType string
}

// Query executes a query within the transaction
func (t *Tx) Query(query DBQuery, args ...any) (*sql.Rows, error) {
	sqlQuery := query.GetQuery(t.DBType)
	return t.Tx.Query(sqlQuery, args...)
}

// Exec executes a command within the transaction
func (t *Tx) Exec(query DBQuery, args ...any) (sql.Result, error) {
	sqlQuery := query.GetQuery(t.DBType)
	return t.Tx.Exec(sqlQuery, args...)
}

// Commit commits the transaction
func (t *Tx) Commit() error {
	if err := t.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (t *Tx) Rollback() error {
	if err := t.Tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
