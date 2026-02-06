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

package utils

import (
	"fmt"
	"strings"
)

// BuildPaginationQuery adds LIMIT and OFFSET clauses to a query.
func BuildPaginationQuery(baseQuery string, limit, offset int) string {
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", baseQuery, limit, offset)
}

// BuildOrderByQuery adds ORDER BY clause to a query.
func BuildOrderByQuery(baseQuery string, orderBy string, ascending bool) string {
	direction := "ASC"
	if !ascending {
		direction = "DESC"
	}
	return fmt.Sprintf("%s ORDER BY %s %s", baseQuery, orderBy, direction)
}

// ConvertToPostgresParams converts ? placeholders to $1, $2, etc. for PostgreSQL.
func ConvertToPostgresParams(query string) string {
	paramIndex := 1
	var result strings.Builder
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			result.WriteString(fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		} else {
			result.WriteByte(query[i])
		}
	}
	return result.String()
}
