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

// DBQueryInterface defines the interface for database queries.
type DBQueryInterface interface {
	GetID() string
	GetQuery(dbType string) string
}

var _ DBQueryInterface = (*DBQuery)(nil)

// DBQuery represents database queries with an identifier and the SQL query string.
// It supports multiple database types (MySQL, PostgreSQL, SQLite) with database-specific variants.
type DBQuery struct {
	// ID is the unique identifier for the query.
	ID string `json:"id"`
	// Query is the default query (MySQL syntax).
	Query string `json:"query"`
	// PostgresQuery is the PostgreSQL-specific query variant.
	PostgresQuery string `json:"postgres_query,omitempty"`
	// SQLiteQuery is the SQLite-specific query variant.
	SQLiteQuery string `json:"sqlite_query,omitempty"`
}

// GetID returns the unique identifier for the query.
func (d *DBQuery) GetID() string {
	return d.ID
}

// GetQuery returns the appropriate query for the specified database type.
// If a database-specific query is not available, it falls back to the default query.
func (d *DBQuery) GetQuery(dbType string) string {
	switch dbType {
	case "postgres", "postgresql":
		if d.PostgresQuery != "" {
			return d.PostgresQuery
		}
	case "sqlite", "sqlite3":
		if d.SQLiteQuery != "" {
			return d.SQLiteQuery
		}
	}
	// Fall back to the default query (MySQL)
	return d.Query
}
