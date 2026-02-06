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

package testutils

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ServerBinary = "../../bin/consent-server"
	ConfigPath   = "repository/conf/deployment.yaml" // Relative to tests/integration
)

var serverCmd *exec.Cmd

type ServerConfig struct {
	Server struct {
		Hostname string `yaml:"hostname"`
		Port     int    `yaml:"port"`
	} `yaml:"server"`
}

// GetServerPort reads the port from deployment.yaml
func GetServerPort() string {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		// Fallback to default port if config file not found
		fmt.Printf("Warning: Could not read config file: %v, using default port 3000\n", err)
		return "3000"
	}

	var config ServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("Warning: Could not parse config file: %v, using default port 3000\n", err)
		return "3000"
	}

	if config.Server.Port == 0 {
		return "3000"
	}

	return fmt.Sprintf("%d", config.Server.Port)
}

// BuildServer checks if the consent-server binary exists
// The binary should be built using ./build.sh build from the project root
func BuildServer() error {
	fmt.Println("Checking for consent server binary...")

	// Check if binary exists
	if _, err := os.Stat(ServerBinary); os.IsNotExist(err) {
		return fmt.Errorf("server binary not found at %s. Please run './build.sh build' from project root", ServerBinary)
	}

	fmt.Println("✓ Found server binary at", ServerBinary)
	return nil
}

// SetupDatabase runs database migration scripts
func SetupDatabase() error {
	fmt.Println("Cleaning and setting up test database configured in tests/integration/repository/conf/deployment.yaml...")

	// Read database config
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Database struct {
			Consent struct {
				Hostname string `yaml:"hostname"`
				Port     int    `yaml:"port"`
				Database string `yaml:"database"`
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"consent"`
		} `yaml:"database"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	dbConfig := config.Database.Consent

	// Build mysql command to run schema
	schemaFile := "../../consent-server/dbscripts/db_schema_mysql.sql"

	// Check if schema file exists
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", schemaFile)
	}

	// Run mysql command to create schema
	// First disable foreign key checks, drop all tables, run the schema, then re-enable
	// This ensures a clean database state for testing
	sqlScript := fmt.Sprintf("SET FOREIGN_KEY_CHECKS=0; DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; source %s; SET FOREIGN_KEY_CHECKS=1;",
		dbConfig.Database, dbConfig.Database, dbConfig.Database, schemaFile)
	cmd := exec.Command("mysql",
		"-h", dbConfig.Hostname,
		"-P", fmt.Sprintf("%d", dbConfig.Port),
		"-u", dbConfig.User,
		fmt.Sprintf("-p%s", dbConfig.Password),
		"-e", sqlScript,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run database schema: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// StartServer starts the consent-server in background
func StartServer() error {
	fmt.Println("Starting consent server...")
	cmd := exec.Command("./consent-server")
	cmd.Dir = "../../bin" // Run from bin directory where config files are located
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables for test mode
	port := GetServerPort()
	cmd.Env = append(os.Environ(),
		"SERVER_PORT="+port,
		"LOG_LEVEL=debug",
	)

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	serverCmd = cmd
	return nil
}

// StopServer gracefully stops the consent-server
func StopServer() error {
	if serverCmd == nil || serverCmd.Process == nil {
		return nil
	}

	fmt.Println("Stopping server...")

	// Send interrupt signal
	err := serverCmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Wait for process to exit
	_, err = serverCmd.Process.Wait()
	return err
}

// WaitForServer waits for the server to be ready
func WaitForServer() error {
	fmt.Println("Waiting for server to be ready...")
	port := GetServerPort()
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:" + port + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println("✓ Server is ready!")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("server did not start within timeout")
}
