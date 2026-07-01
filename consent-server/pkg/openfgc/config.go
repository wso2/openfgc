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

package openfgc

import (
	"time"

	"github.com/wso2/openfgc/consent-server/internal/system/config"
)

// Config is the top-level library configuration.
type Config struct {
	DB                 DBConfig
	StatusMappings     *StatusMappings
	AuthStatusMappings *AuthStatusMappings
}

// DBConfig describes the database openfgc should connect to.
type DBConfig struct {
	Type            string
	Hostname        string
	Port            int
	User            string
	Password        string
	Database        string
	Path            string
	SSLMode         string
	Options         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// StatusMappings overrides consent lifecycle status strings; defaults apply when nil.
type StatusMappings struct {
	Active   string
	Expired  string
	Revoked  string
	Created  string
	Rejected string
}

// AuthStatusMappings overrides authorization lifecycle status strings; defaults apply when nil.
type AuthStatusMappings struct {
	Approved      string
	Rejected      string
	Created       string
	SystemExpired string
	SystemRevoked string
}

func defaultStatusMappings() config.ConsentStatusMappings {
	return config.ConsentStatusMappings{
		ActiveStatus:   "ACTIVE",
		ExpiredStatus:  "EXPIRED",
		RevokedStatus:  "REVOKED",
		CreatedStatus:  "CREATED",
		RejectedStatus: "REJECTED",
	}
}

func defaultAuthStatusMappings() config.AuthStatusMappings {
	return config.AuthStatusMappings{
		ApprovedState:      "APPROVED",
		RejectedState:      "REJECTED",
		CreatedState:       "CREATED",
		SystemExpiredState: "SYS_EXPIRED",
		SystemRevokedState: "SYS_REVOKED",
	}
}

func (c Config) toInternal() *config.Config {
	sm := defaultStatusMappings()
	if c.StatusMappings != nil {
		sm = config.ConsentStatusMappings{
			ActiveStatus:   c.StatusMappings.Active,
			ExpiredStatus:  c.StatusMappings.Expired,
			RevokedStatus:  c.StatusMappings.Revoked,
			CreatedStatus:  c.StatusMappings.Created,
			RejectedStatus: c.StatusMappings.Rejected,
		}
	}
	asm := defaultAuthStatusMappings()
	if c.AuthStatusMappings != nil {
		asm = config.AuthStatusMappings{
			ApprovedState:      c.AuthStatusMappings.Approved,
			RejectedState:      c.AuthStatusMappings.Rejected,
			CreatedState:       c.AuthStatusMappings.Created,
			SystemExpiredState: c.AuthStatusMappings.SystemExpired,
			SystemRevokedState: c.AuthStatusMappings.SystemRevoked,
		}
	}
	return &config.Config{
		// Placeholder to satisfy validateConfig; no HTTP server runs in library mode.
		Server: config.ServerConfig{Hostname: "library", Port: 1},
		Database: config.DatabasesConfig{
			Consent: config.DatabaseConfig{
				Type:            c.DB.Type,
				Hostname:        c.DB.Hostname,
				Port:            c.DB.Port,
				User:            c.DB.User,
				Password:        c.DB.Password,
				Database:        c.DB.Database,
				Path:            c.DB.Path,
				SSLMode:         c.DB.SSLMode,
				Options:         c.DB.Options,
				MaxOpenConns:    c.DB.MaxOpenConns,
				MaxIdleConns:    c.DB.MaxIdleConns,
				ConnMaxLifetime: c.DB.ConnMaxLifetime,
			},
		},
		Consent: config.ConsentConfig{
			StatusMappings:     sm,
			AuthStatusMappings: asm,
		},
	}
}
