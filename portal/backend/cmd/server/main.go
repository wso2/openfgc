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

// Package main starts the OpenFGC portal backend BFF service.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wso2/openfgc/portal/backend/internal/config"
	"github.com/wso2/openfgc/portal/backend/internal/logger"
	"github.com/wso2/openfgc/portal/backend/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level)
	if cfg.Proxy.PlaceholderModeEnabled {
		log.Warn("placeholder identity mode is enabled; do not use in production")
	}
	handler, err := router.New(log, *cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize router: %v\n", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	serveErrCh := make(chan error, 1)

	go func() {
		log.Info("starting OpenFGC portal backend server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	exitCode := 0

	select {
	case <-ctx.Done():
	case err := <-serveErrCh:
		log.Error("server stopped unexpectedly", "error", err)
		exitCode = 1
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	log.Info("shutting down OpenFGC portal backend server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		exitCode = 1
	}

	time.Sleep(50 * time.Millisecond)
	log.Info("OpenFGC portal backend server stopped")

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
