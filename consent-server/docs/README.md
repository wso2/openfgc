
# Consent Management API — Project Summary

This repository contains the Consent Management service (Go + Gin) and a suite of integration tests that exercise consent create/read/update/delete, authorization status transitions, and validation logic.

**Purpose of this doc**: give a concise overview of the project, where to find important pieces, how to run the service/tests locally, and the recent status-transition bug fix summary.

## Project Overview
- Language: Go
- Web framework: Gin
- Testing: Go `testing` + `testify` (integration tests under `integration-tests/api`)
- Database: MySQL (DB schema in `dbscripts/db_schema_mysql.sql`)

## Important Directories
- `cmd/server` — service entrypoint
- `internal/handlers` — HTTP handlers for consent APIs
- `internal/service` — business logic and orchestration
- `internal/dao` — database access objects
- `internal/models` — API/DB models and constants
- `integration-tests/api` — integration tests for different endpoints (consents, purposes, etc.)
- `dbscripts` — SQL schema and helper scripts

## Recent Bug Fix (Status Transition)
Issue: When a consent had a non-ACTIVE status (for example `REVOKED` or `REJECTED`) and its `validityTime` passed, calling GET/PUT/VALIDATE incorrectly transitioned the consent to `EXPIRED`.

Fix: expiry checks now only apply to consents in the `ACTIVE` state. The GET (read), PUT (update), and POST (validate) handlers were updated to use the `IsActiveStatus()` check before applying expiration logic. This preserves terminal states like `REVOKED` and `REJECTED`.

Files changed: handler logic updates in `internal/handlers/consent_handler.go`.

Tests: status transition tests were added/updated and moved into existing test files for clarity (see Test organization below).

## Test Organization (where to find the status tests)
- `integration-tests/api/consent/consent_revoke_test.go`
  - `TestRevokedConsentDoesNotBecomeExpired` — creates an ACTIVE consent, revokes it, updates its validity to a past time, and verifies GET does not change the status from `REVOKED` to `EXPIRED`.
- `integration-tests/api/consent/read_test.go`
  - `TestRejectedConsentDoesNotBecomeExpired` — verifies `REJECTED` consents remain `REJECTED` on GET even if validity has passed.
  - `TestActiveConsentStillBecomesExpired` — verifies ACTIVE consents still become `EXPIRED` when validity has passed.

Note: `integration-tests/api/consent/status_transition_test.go` was removed after migrating its tests into the appropriate files.

## Running Locally

Prerequisites
- Go >= 1.21
- MySQL database (set up using `dbscripts/db_schema_mysql.sql`)

Environment
- Set `org-id` and `client-id` headers when making requests in tests; the tests set `TEST_ORG` / `TEST_CLIENT` by default.

### Quick Start

```bash
# Build the server
./build.sh build

# Start the server normally
./start.sh

# Or start in debug mode for development
./start.sh --debug
```

### Build & Run Commands

```bash
# Build for current platform
./build.sh build

# Build for specific platform
./build.sh build linux amd64

# Create distribution package
./build.sh package

# Run tests
./build.sh test_unit           # Unit tests
./build.sh test_integration    # Integration tests
./build.sh test                # All tests

# Clean build artifacts
./build.sh clean
```

### Start Script Options

```bash
# Normal start
./start.sh

# Debug mode (with Delve remote debugger)
./start.sh --debug

# Custom ports
./start.sh --port 9090 --debug-port 3456

# Help
./start.sh --help
```

Run integration tests (consent-specific)

```bash
cd /Users/hasithan/Projects/go/consent-mgt-v1
# Run consent integration tests (no cache)
go test -v ./integration-tests/api/consent -count=1
```

Run all integration tests

```bash
# Using build script (recommended)
./build.sh test_integration

# Or directly
go test -v ./integration-tests/... -count=1
```

Run unit tests (package-level)

```bash
# Using build script
./build.sh test_unit

# Or directly
go test ./... -run TestName -count=1
```

Database setup (quick)

1. Create a MySQL database (example name `consent_mgt_dev`).
2. Import schema:

```bash
mysql -u root -p consent_mgt_dev < dbscripts/db_schema_mysql.sql
```

Adjust connection settings in `bin/repository/conf/deployment.yaml` or set `CONFIG_PATH` environment variable.

## Contributing / Review Notes
- Place tests in files that reflect the operation under test (create/revoke/read/update).
- Keep tests focused; if a test spans multiple flows (create→revoke→update→read) put it in the file matching the flow's primary operation — e.g., revoke tests in `consent_revoke_test.go`.
- Include `defer CleanupTestData(...)` where appropriate to avoid leaking test data.

## Changelog / Next Steps
- I can prepare a concise changelog entry and/or a PR description summarizing the handler fix and test reorganization if you'd like.

## Contact
- Ping the maintainer or open an issue/PR in this repository for follow-up work.

