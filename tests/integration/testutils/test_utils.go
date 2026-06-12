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
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// dbType holds the database type for the current test run ("sqlite" or "mysql").
// Populated from the DB_TYPE environment variable; defaults to "mysql".
var dbType = func() string {
	if t := os.Getenv("DB_TYPE"); t != "" {
		return t
	}
	return "mysql"
}()

const (
	ServerBinary        = "../../target/server/consent-server"
	ServerBinaryWindows = "../../target/server/consent-server.exe"
)

// ConfigPath returns the integration test deployment config path for the active dbType.
// Relative to the tests/integration/ directory.
var ConfigPath = func() string {
	if dbType == "sqlite" {
		return "repository/conf/deployment-sqlite.yaml"
	}
	return "repository/conf/deployment.yaml"
}()

var serverCmd *exec.Cmd

type ServerConfig struct {
	Server struct {
		Hostname string `yaml:"hostname"`
		Port     int    `yaml:"port"`
	} `yaml:"server"`
}

// getServerBinary returns the platform-specific binary path and executable name.
func getServerBinary() (binaryPath string, binaryName string) {
	if runtime.GOOS == "windows" {
		return ServerBinaryWindows, "consent-server.exe"
	}
	return ServerBinary, "./consent-server"
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

// BuildServer checks if the consent-server binary exists.
// The binary should be built using ./build.sh build from the project root.
func BuildServer() error {
	fmt.Println("Checking for consent server binary...")

	binaryPath, _ := getServerBinary()
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("server binary not found at %s. Please run './build.sh build' from project root", binaryPath)
	}

	fmt.Println("✓ Found server binary at", binaryPath)
	return nil
}

// SetupDatabase cleans and re-initialises the test database.
// It branches on dbType: "sqlite" uses the sqlite3 CLI, "mysql" uses the mysql CLI.
func SetupDatabase() error {
	fmt.Printf("Setting up test database (type=%s)...\n", dbType)

	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Database struct {
			Consent struct {
				Type     string `yaml:"type"`
				Path     string `yaml:"path"`
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

	supported := map[string]bool{"mysql": true, "sqlite": true}
	if !supported[dbType] {
		return fmt.Errorf("unsupported DB_TYPE %q: must be one of mysql, sqlite", dbType)
	}
	if dbConfig.Type != dbType {
		return fmt.Errorf("DB_TYPE env var is %q but config file has database.consent.type=%q", dbType, dbConfig.Type)
	}

	if dbType == "sqlite" {
		// Resolve the db file path: the server runs from target/server/,
		// so prepend that to the relative path from the config.
		serverDir := "../../target/server"
		dbPath := filepath.Join(serverDir, dbConfig.Path)
		schemaFile := "../../consent-server/dbscripts/db_schema_sqlite.sql"
		return initSQLiteDB(dbPath, schemaFile)
	}

	// MySQL: drop and recreate the database, then apply the schema.
	schemaFile := "../../consent-server/dbscripts/db_schema_mysql.sql"
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", schemaFile)
	}

	sqlScript := fmt.Sprintf(
		"SET FOREIGN_KEY_CHECKS=0; DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; source %s; SET FOREIGN_KEY_CHECKS=1;",
		dbConfig.Database, dbConfig.Database, dbConfig.Database, schemaFile,
	)
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

// initSQLiteDB creates a fresh SQLite database from a schema file using the sqlite3 CLI.
// Any existing database file is deleted first to guarantee a clean state.
func initSQLiteDB(dbPath, schemaPath string) error {
	// Fail fast if the sqlite3 CLI is not available.
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return fmt.Errorf("sqlite3 CLI not found in PATH: please install sqlite3 to run SQLite integration tests")
	}

	absSchemaPath, err := filepath.Abs(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to resolve schema path: %w", err)
	}
	if _, err := os.Stat(absSchemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", absSchemaPath)
	}

	absDbPath, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("failed to resolve db path: %w", err)
	}

	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(absDbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Remove old db and WAL files for a clean start.
	for _, f := range []string{absDbPath, absDbPath + "-shm", absDbPath + "-wal"} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing database file %s: %w", f, err)
		}
	}

	// Apply schema by piping the file into sqlite3 via stdin.
	schemaFile, err := os.Open(absSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to open schema file: %w", err)
	}
	defer schemaFile.Close()

	cmd := exec.Command("sqlite3", absDbPath)
	cmd.Stdin = schemaFile
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply SQLite schema: %w", err)
	}

	// Enable WAL mode.
	cmd = exec.Command("sqlite3", absDbPath, "PRAGMA journal_mode=WAL;")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	fmt.Printf("✓ SQLite database initialized: %s\n", absDbPath)
	return nil
}

// StartServer starts the consent-server in background
func StartServer() error {
	fmt.Println("Starting consent server...")

	binaryPath, _ := getServerBinary() // Use binaryPath, not binaryName

	// Convert to absolute path to avoid working directory confusion
	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	cmd := exec.Command(absBinaryPath) // Use full path
	cmd.Dir = "../../target/server"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables for test mode
	port := GetServerPort()
	cmd.Env = append(os.Environ(),
		"SERVER_PORT="+port,
		"LOG_LEVEL=debug",
	)

	err = cmd.Start()
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
