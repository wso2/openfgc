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

package openfgc_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/wso2/openfgc/consent-server/pkg/openfgc"
)

func TestSmokeLibraryEndToEnd(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "consent.db")

	applySchema(t, dbPath)

	client, err := openfgc.New(openfgc.Config{
		DB: openfgc.DBConfig{
			Type:         "sqlite",
			Path:         dbPath,
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
	})
	if err != nil {
		t.Fatalf("openfgc.New: %v", err)
	}
	// Register shutdown immediately so a later t.Fatalf can't leak the
	// package-level single-client guard into subsequent tests.
	t.Cleanup(func() {
		if err := client.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	})

	ctx := context.Background()
	out, sverr := client.Elements().CreateElementsInBatch(
		ctx,
		[]openfgc.CreateElementInput{
			{
				Name:        "email",
				Namespace:   "default",
				Description: ptrString("user email address"),
				Type:        "basic",
			},
		},
		"org-1",
	)
	if sverr != nil {
		t.Fatalf("CreateElementsInBatch: %v", sverr)
	}
	if out == nil || len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %+v", out)
	}
	if out.Results[0].Status != "SUCCESS" {
		reason := ""
		if out.Results[0].Error != nil {
			reason = *out.Results[0].Error
		}
		t.Fatalf("expected SUCCESS, got status=%q error=%q", out.Results[0].Status, reason)
	}

	list, sverr := client.Elements().ListElements(ctx, "org-1", openfgc.ElementListFilter{Limit: 10})
	if sverr != nil {
		t.Fatalf("ListElements: %v", sverr)
	}
	if list == nil || len(list.Data) != 1 {
		t.Fatalf("expected 1 element listed, got %+v", list)
	}
}

func TestSingleActiveClientGuard(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "consent.db")
	applySchema(t, dbPath)

	cfg := openfgc.Config{
		DB: openfgc.DBConfig{
			Type:         "sqlite",
			Path:         dbPath,
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
	}

	client, err := openfgc.New(cfg)
	if err != nil {
		t.Fatalf("first New: %v", err)
	}

	// A second Client must be refused while the first is still active.
	if _, err := openfgc.New(cfg); err == nil {
		t.Fatal("expected second New to fail while a Client is active, got nil error")
	}

	// After Shutdown, a fresh Client must be constructible again.
	if err := client.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	client2, err := openfgc.New(cfg)
	if err != nil {
		t.Fatalf("New after Shutdown: %v", err)
	}
	if err := client2.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}

func applySchema(t *testing.T, dbPath string) {
	t.Helper()

	schemaPath := filepath.Join("..", "..", "dbscripts", "db_schema_sqlite.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite for schema: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if _, err := db.Exec(string(schemaBytes)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func ptrString(s string) *string { return &s }
