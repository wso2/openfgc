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

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/wso2/openfgc/tests/integration/testutils"
)

func main() {
	// Step 1: Check for pre-built server binary
	err := testutils.BuildServer()
	if err != nil {
		fmt.Printf("Failed to find server binary: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Setup test database
	err = testutils.SetupDatabase()
	if err != nil {
		fmt.Printf("Failed to setup database: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Database initialized")

	// Step 3: Start server
	err = testutils.StartServer()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
	defer testutils.StopServer()

	// Step 4: Wait for server to be ready
	time.Sleep(2 * time.Second) // Give server a moment to start
	err = testutils.WaitForServer()
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		testutils.StopServer()
		os.Exit(1)
	}

	// Step 5: Run tests
	fmt.Println("\nRunning tests...")
	err = runTests()
	if err != nil {
		fmt.Printf("Tests failed: %v\n", err)
		testutils.StopServer()
		os.Exit(1)
	}

	fmt.Println("\n✓ All tests completed successfully!")
}

func runTests() error {
	// Run all test packages
	packages := []string{
		"./consentelement",
		"./consentpurpose",
		"./consent",
	}

	for _, pkg := range packages {
		fmt.Printf("\nRunning tests in %s...\n", pkg)
		cmd := exec.Command("go", "test", "-v", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tests failed in package %s: %w", pkg, err)
		}
	}

	return nil
}
