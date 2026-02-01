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

// Package provider provides functionality for managing database connections and clients.
package provider

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/wso2/consent-management-api/internal/system/config"
	"github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/transaction"
	"github.com/wso2/consent-management-api/internal/system/log"
)

// transactionDBWrapper wraps model.DB to implement transaction.DBInterface
type transactionDBWrapper struct {
	db *model.DB
}

// BeginTx implements transaction.DBInterface
func (w *transactionDBWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return w.db.BeginTxContext(ctx, opts)
}

// DBProviderInterface defines the interface for getting database clients.
type DBProviderInterface interface {
	GetConsentDBClient() (DBClientInterface, error)
	GetConsentDBTransactioner() (transaction.Transactioner, error)
}

// DBProviderCloser is a separate interface for closing the provider.
// Only the lifecycle manager should use this interface.
type DBProviderCloser interface {
	Close() error
}

// dbProvider is the implementation of DBProviderInterface.
type dbProvider struct {
	consentClient        DBClientInterface
	consentTransactioner transaction.Transactioner
	consentMutex         sync.RWMutex
	db                   *model.DB
}

var (
	instance *dbProvider
	once     sync.Once
)

// initDBProvider initializes the singleton instance of DBProvider.
func initDBProvider() {
	once.Do(func() {
		instance = &dbProvider{}
		instance.initializeClient()
	})
}

// GetDBProvider returns the instance of DBProvider.
func GetDBProvider() DBProviderInterface {
	initDBProvider()
	return instance
}

// GetDBProviderCloser returns the DBProvider with closing capability.
// This should only be called from the main lifecycle manager.
func GetDBProviderCloser() DBProviderCloser {
	initDBProvider()
	return instance
}

// GetConsentDBClient returns a database client for consent datasource.
// Not required to close the returned client manually since it manages its own connection pool.
func (d *dbProvider) GetConsentDBClient() (DBClientInterface, error) {
	d.consentMutex.RLock()
	if d.consentClient != nil {
		client := d.consentClient
		d.consentMutex.RUnlock()
		return client, nil
	}
	d.consentMutex.RUnlock()

	// Initialize client if not already done
	d.consentMutex.Lock()
	defer d.consentMutex.Unlock()

	// Double-check after acquiring write lock
	if d.consentClient != nil {
		return d.consentClient, nil
	}

	// Initialize now
	if err := d.initializeClientLocked(); err != nil {
		return nil, err
	}

	return d.consentClient, nil
}

// GetConsentDBTransactioner returns a transactioner for consent datasource.
func (d *dbProvider) GetConsentDBTransactioner() (transaction.Transactioner, error) {
	d.consentMutex.RLock()
	if d.consentTransactioner != nil {
		t := d.consentTransactioner
		d.consentMutex.RUnlock()
		return t, nil
	}
	d.consentMutex.RUnlock()

	// Initialize transactioner if not already done
	d.consentMutex.Lock()
	defer d.consentMutex.Unlock()

	// Double-check after acquiring write lock
	if d.consentTransactioner != nil {
		return d.consentTransactioner, nil
	}

	// Initialize now
	if err := d.initializeClientLocked(); err != nil {
		return nil, err
	}

	return d.consentTransactioner, nil
}

// initializeClient initializes the database client at startup (called from once.Do).
func (d *dbProvider) initializeClient() {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBProvider"))

	d.consentMutex.Lock()
	defer d.consentMutex.Unlock()

	if err := d.initializeClientLocked(); err != nil {
		logger.Error("Failed to initialize consent database client", log.Error(err))
	}
}

// initializeClientLocked initializes the database client (must be called with lock held).
func (d *dbProvider) initializeClientLocked() error {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBProvider"))

	// Get database configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	dbConfig := &cfg.Database.Consent

	// Initialize database
	db, err := initializeDB(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	d.db = db
	d.consentClient = NewDBClient(d.db, "mysql")
	d.consentTransactioner = transaction.NewTransactioner(&transactionDBWrapper{db: d.db})
	logger.Debug("Consent DB client initialized")

	return nil
}

// Close closes the database connections. This should only be called by the lifecycle manager during shutdown.
func (d *dbProvider) Close() error {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBProvider"))
	logger.Debug("Closing database connections")

	d.consentMutex.Lock()
	defer d.consentMutex.Unlock()

	if d.db != nil {
		if err := d.db.Close(); err != nil {
			logger.Error("Failed to close database connection", log.Error(err))
			return fmt.Errorf("failed to close database: %w", err)
		}
		d.db = nil
	}

	d.consentClient = nil
	logger.Debug("Database connections closed")

	return nil
}

// initializeDB creates and initializes a new database connection.
func initializeDB(cfg *config.DatabaseConfig) (*model.DB, error) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "Database"))
	dsn := cfg.GetDSN()

	logger.Info("Connecting to database...",
		log.String("hostname", cfg.Hostname),
		log.Int("port", cfg.Port),
		log.String("database", cfg.Database))

	// Open database connection
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to database")

	return &model.DB{DB: db, DBType: cfg.Type}, nil
}
