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

// Package main is the entry point for starting the consent server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wso2/openfgc/internal/system/config"
	"github.com/wso2/openfgc/internal/system/database/provider"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/middleware"
)

func main() {
	logger := log.GetLogger()

	// Load configuration and setup logging
	cfg := initializeConfiguration(logger)

	// Setup HTTP server
	server := setupHTTPServer(cfg, logger)

	// Start server
	startServer(server, cfg, logger)

	// Wait for shutdown signal
	waitForShutdown(server, logger)
}

// initializeConfiguration loads config and sets up log level
func initializeConfiguration(logger *log.Logger) *config.Config {

	// Priority: CONFIG_PATH env var > repository/conf/deployment.yaml > cmd/server/repository/conf/deployment.yaml
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", log.Error(err))
	}

	logger.Info("Configuration loaded successfully", log.String("config_path", configPath))

	// Update log level from configuration
	if cfg.Logging.Level != "" {
		if err := log.SetLogLevel(cfg.Logging.Level); err != nil {
			logger.Error("Failed to set log level from configuration", log.Error(err))
		} else {
			logger.Debug("Log level updated from configuration", log.String("level", cfg.Logging.Level))
		}
	}

	return cfg
}

// setupHTTPServer creates and configures the HTTP server
func setupHTTPServer(cfg *config.Config, logger *log.Logger) *http.Server {
	// Create HTTP mux
	mux := http.NewServeMux()

	// Register all services
	registerServices(mux)

	// Wrap with correlation ID middleware
	httpHandler := middleware.WrapWithCorrelationID(mux)

	// Configure HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Hostname, cfg.Server.Port)
	server := &http.Server{
		Addr:           serverAddr,
		Handler:        httpHandler,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Log server configuration
	logger.Info("HTTP server configured",
		log.String("address", serverAddr),
		log.Int("port", cfg.Server.Port),
		log.String("read_timeout", cfg.Server.ReadTimeout.String()),
		log.String("write_timeout", cfg.Server.WriteTimeout.String()),
		log.String("idle_timeout", cfg.Server.IdleTimeout.String()),
	)

	return server
}

// startServer starts the HTTP server in a goroutine
func startServer(server *http.Server, cfg *config.Config, logger *log.Logger) {
	go func() {
		logger.Info("Starting HTTP server...",
			log.String("hostname", cfg.Server.Hostname),
			log.Int("port", cfg.Server.Port),
			log.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", log.Error(err))
		}
	}()

	logger.Info("✓ Server is running", log.String("address", server.Addr))
	logger.Info("Press Ctrl+C to stop the server")
}

// waitForShutdown waits for interrupt signal and gracefully shuts down
func waitForShutdown(server *http.Server, logger *log.Logger) {
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", log.Error(err))
	}

	// Unregister services
	unregisterServices()
	logger.Info("Services unregistered")

	closeDatabase(logger)
	logger.Info("Server exited gracefully")
}

// closeDatabase closes database connections
func closeDatabase(logger *log.Logger) {
	// Close database connections
	dbCloser := provider.GetDBProviderCloser()
	if err := dbCloser.Close(); err != nil {
		logger.Error("Error closing database connections", log.Error(err))
	} else {
		logger.Debug("Database connections closed successfully")
	}
}
