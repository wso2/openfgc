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

// Package config provides structures and functions for loading and managing server configurations.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wso2/openfgc/internal/system/log"
	"gopkg.in/yaml.v3"
)

// globalConfig holds the application configuration
var globalConfig *Config

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Database DatabasesConfig `yaml:"database"`
	Logging  LoggingConfig   `yaml:"logging"`
	Consent  ConsentConfig   `yaml:"consent"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Hostname     string        `yaml:"hostname"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
}

// DatabasesConfig holds all database configurations
type DatabasesConfig struct {
	Consent DatabaseConfig `yaml:"consent"`
}

// DatabaseConfig holds individual database configuration
type DatabaseConfig struct {
	Type            string        `yaml:"type"`
	Hostname        string        `yaml:"hostname"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	Path            string        `yaml:"path"`
	SSLMode         string        `yaml:"sslmode"`
	Options         string        `yaml:"options"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// ConsentStatus represents a typed consent status
type ConsentStatus string

// AuthStatus represents a typed authorization status
type AuthStatus string

// ConsentConfig holds consent-related configuration
type ConsentConfig struct {
	StatusMappings     ConsentStatusMappings `yaml:"status_mappings"`
	AuthStatusMappings AuthStatusMappings    `yaml:"auth_status_mappings"`
}

// ConsentStatusMappings holds the mapping of specific consent lifecycle states
type ConsentStatusMappings struct {
	ActiveStatus   string `yaml:"active_status"`
	ExpiredStatus  string `yaml:"expired_status"`
	RevokedStatus  string `yaml:"revoked_status"`
	CreatedStatus  string `yaml:"created_status"`
	RejectedStatus string `yaml:"rejected_status"`
}

// AuthStatusMappings holds the mapping of authorization resource lifecycle states
type AuthStatusMappings struct {
	ApprovedState      string `yaml:"approved_state"`
	RejectedState      string `yaml:"rejected_state"`
	CreatedState       string `yaml:"created_state"`
	SystemExpiredState string `yaml:"system_expired_state"`
	SystemRevokedState string `yaml:"system_revoked_state"`
}

// GetActiveConsentStatus returns the typed active status from config
func (c *ConsentConfig) GetActiveConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.ActiveStatus)
}

// GetExpiredConsentStatus returns the typed expired status from config
func (c *ConsentConfig) GetExpiredConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.ExpiredStatus)
}

// GetRevokedConsentStatus returns the typed revoked status from config
func (c *ConsentConfig) GetRevokedConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.RevokedStatus)
}

// GetCreatedConsentStatus returns the typed created status from config
func (c *ConsentConfig) GetCreatedConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.CreatedStatus)
}

// GetRejectedConsentStatus returns the typed rejected status from config
func (c *ConsentConfig) GetRejectedConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.RejectedStatus)
}

// GetApprovedAuthStatus returns the typed approved auth status from config
func (c *ConsentConfig) GetApprovedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.ApprovedState)
}

// GetRejectedAuthStatus returns the typed rejected auth status from config
func (c *ConsentConfig) GetRejectedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.RejectedState)
}

// GetCreatedAuthStatus returns the typed created auth status from config
func (c *ConsentConfig) GetCreatedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.CreatedState)
}

// GetSystemExpiredAuthStatus returns the typed system expired auth status from config
func (c *ConsentConfig) GetSystemExpiredAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.SystemExpiredState)
}

// GetSystemRevokedAuthStatus returns the typed system revoked auth status from config
func (c *ConsentConfig) GetSystemRevokedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.SystemRevokedState)
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	logger := log.GetLogger()
	logger.Debug("Loading configuration", log.String("config_path", configPath))

	// Determine config file path
	var finalPath string
	if configPath != "" {
		finalPath = configPath
	} else {
		// Default configuration lookup order:
		// 1. ./repository/conf/deployment.yaml (production - relative to binary)
		// 2. ./cmd/server/repository/conf/deployment.yaml (development)
		paths := []string{
			"./repository/conf/deployment.yaml",
			"./cmd/server/repository/conf/deployment.yaml",
			"../repository/conf/deployment.yaml",
			"./deployment.yaml",
		}

		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				finalPath = path
				break
			}
		}

		if finalPath == "" {
			return nil, fmt.Errorf("no configuration file found in default paths")
		}
	}

	// Read the config file
	finalPath = filepath.Clean(finalPath)
	data, err := os.ReadFile(finalPath)
	if err != nil {
		logger.Error("Failed to read config file", log.Error(err))
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables
	data, err = substituteEnvironmentVariables(data)
	if err != nil {
		logger.Error("Failed to substitute environment variables", log.Error(err))
		return nil, fmt.Errorf("failed to substitute environment variables: %w", err)
	}

	logger.Info("Config file loaded", log.String("file", finalPath))

	// Unmarshal config
	var config Config
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		logger.Error("Failed to unmarshal config", log.Error(err))
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		logger.Error("Config validation failed", log.Error(err))
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	globalConfig = &config
	logger.Debug("Configuration loaded and validated successfully",
		log.String("server_port", fmt.Sprintf("%d", config.Server.Port)),
		log.String("db_host", config.Database.Consent.Hostname),
	)
	return &config, nil
}

// substituteEnvironmentVariables replaces ${VAR_NAME} patterns with environment variable values
func substituteEnvironmentVariables(data []byte) ([]byte, error) {
	content := string(data)

	// Find all ${...} patterns using a moving index
	pos := 0
	for pos < len(content) {
		start := strings.Index(content[pos:], "${")
		if start == -1 {
			break
		}
		start += pos
		end := strings.Index(content[start:], "}")
		if end == -1 {
			return nil, fmt.Errorf("unclosed environment variable substitution at position %d", start)
		}
		end += start

		// Extract variable name
		varName := content[start+2 : end]

		// Get environment variable value
		varValue := os.Getenv(varName)

		// Replace the pattern with the value
		content = content[:start] + varValue + content[end+1:]

		// Move position past the replaced value to avoid re-expansion
		pos = start + len(varValue)
	}

	return []byte(content), nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	switch config.Database.Consent.Type {
	case "sqlite":
		if config.Database.Consent.Path == "" {
			return fmt.Errorf("database path is required for SQLite")
		}
	case "postgres":
		if config.Database.Consent.Hostname == "" {
			return fmt.Errorf("database hostname is required")
		}
		if config.Database.Consent.Database == "" {
			return fmt.Errorf("database name is required")
		}
	default: // mysql and empty (defaults to mysql)
		if config.Database.Consent.Hostname == "" {
			return fmt.Errorf("database hostname is required")
		}
		if config.Database.Consent.Database == "" {
			return fmt.Errorf("database name is required")
		}
	}

	// Validate consent status mappings
	if config.Consent.StatusMappings.ActiveStatus == "" {
		return fmt.Errorf("consent active status mapping is required")
	}
	if config.Consent.StatusMappings.ExpiredStatus == "" {
		return fmt.Errorf("consent expired status mapping is required")
	}
	if config.Consent.StatusMappings.RevokedStatus == "" {
		return fmt.Errorf("consent revoked status mapping is required")
	}
	if config.Consent.StatusMappings.CreatedStatus == "" {
		return fmt.Errorf("consent created status mapping is required")
	}
	if config.Consent.StatusMappings.RejectedStatus == "" {
		return fmt.Errorf("consent rejected status mapping is required")
	}

	// Validate auth status mappings
	if config.Consent.AuthStatusMappings.ApprovedState == "" {
		return fmt.Errorf("auth approved status mapping is required")
	}
	if config.Consent.AuthStatusMappings.RejectedState == "" {
		return fmt.Errorf("auth rejected status mapping is required")
	}
	if config.Consent.AuthStatusMappings.CreatedState == "" {
		return fmt.Errorf("auth created status mapping is required")
	}
	if config.Consent.AuthStatusMappings.SystemExpiredState == "" {
		return fmt.Errorf("auth system expired status mapping is required")
	}
	if config.Consent.AuthStatusMappings.SystemRevokedState == "" {
		return fmt.Errorf("auth system revoked status mapping is required")
	}

	return nil
}

// Get returns the global configuration
func Get() *Config {
	return globalConfig
}

// SetGlobal sets the global configuration (for testing purposes)
func SetGlobal(cfg *Config) {
	globalConfig = cfg
}

// GetDSN returns the database connection string appropriate for the configured database type.
// For SQLite it returns the file path with optional query parameters.
// For MySQL (default) it returns the standard TCP DSN.
func (d *DatabaseConfig) GetDSN() string {
	switch d.Type {
	case "sqlite":
		options := d.Options
		if options != "" && options[0] != '?' {
			options = "?" + options
		}
		return d.Path + options
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
			d.Hostname,
			d.Port,
			d.User,
			d.Password,
			d.Database,
		)
		if d.SSLMode != "" {
			dsn += " sslmode=" + d.SSLMode
		}
		if d.Options != "" {
			dsn += " " + d.Options
		}
		return dsn
	default: // mysql
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
			d.User,
			d.Password,
			d.Hostname,
			d.Port,
			d.Database,
		)
	}
}

// GetServerAddress returns the server address in host:port format
func (s *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", s.Hostname, s.Port)
}

// IsStatusAllowed checks if a given status is a valid consent status
func (c *ConsentConfig) IsStatusAllowed(status ConsentStatus) bool {
	return status == c.GetActiveConsentStatus() ||
		status == c.GetExpiredConsentStatus() ||
		status == c.GetRevokedConsentStatus() ||
		status == c.GetCreatedConsentStatus() ||
		status == c.GetRejectedConsentStatus()
}

// IsActiveStatus checks if the given status represents an active consent
func (c *ConsentConfig) IsActiveStatus(status ConsentStatus) bool {
	return status == c.GetActiveConsentStatus()
}

// IsExpiredStatus checks if the given status represents an expired consent
func (c *ConsentConfig) IsExpiredStatus(status ConsentStatus) bool {
	return status == c.GetExpiredConsentStatus()
}

// IsRevokedStatus checks if the given status represents a revoked consent
func (c *ConsentConfig) IsRevokedStatus(status ConsentStatus) bool {
	return status == c.GetRevokedConsentStatus()
}

// IsCreatedStatus checks if the given status represents a created consent
func (c *ConsentConfig) IsCreatedStatus(status ConsentStatus) bool {
	return status == c.GetCreatedConsentStatus()
}

// IsRejectedStatus checks if the given status represents a rejected consent
func (c *ConsentConfig) IsRejectedStatus(status ConsentStatus) bool {
	return status == c.GetRejectedConsentStatus()
}

// IsTerminalStatus checks if the given status is a terminal state (expired or revoked)
func (c *ConsentConfig) IsTerminalStatus(status ConsentStatus) bool {
	return c.IsExpiredStatus(status) || c.IsRevokedStatus(status)
}

// GetAllowedConsentStatuses returns a list of all valid consent statuses
func (c *ConsentConfig) GetAllowedConsentStatuses() []ConsentStatus {
	return []ConsentStatus{
		c.GetCreatedConsentStatus(),
		c.GetActiveConsentStatus(),
		c.GetRejectedConsentStatus(),
		c.GetRevokedConsentStatus(),
		c.GetExpiredConsentStatus(),
	}
}

// IsAuthStatusAllowed checks if a given status is a valid authorization status
func (c *ConsentConfig) IsAuthStatusAllowed(status AuthStatus) bool {
	return status == c.GetCreatedAuthStatus() ||
		status == c.GetApprovedAuthStatus() ||
		status == c.GetRejectedAuthStatus() ||
		status == c.GetSystemExpiredAuthStatus() ||
		status == c.GetSystemRevokedAuthStatus()
}

// GetAllowedAuthStatuses returns a list of all valid authorization statuses
func (c *ConsentConfig) GetAllowedAuthStatuses() []AuthStatus {
	return []AuthStatus{
		c.GetCreatedAuthStatus(),
		c.GetApprovedAuthStatus(),
		c.GetRejectedAuthStatus(),
		c.GetSystemExpiredAuthStatus(),
		c.GetSystemRevokedAuthStatus(),
	}
}
