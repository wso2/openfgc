# OpenFGC - Open Fine-Grained Consent

> An industry-agnostic consent management engine with granular control and complete audit trails.

**OpenFGC** is an open-source API service that enables developers to implement consent management at any level of granularity — from individual data elements to broad purposes. Designed for scale with complete audit trails and lifecycle management, it provides everything needed to track, validate, and audit user consent across your applications.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Version](https://img.shields.io/badge/version-v0.5.0-green.svg)](./version.txt)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go)](https://go.dev)
[![MySQL](https://img.shields.io/badge/MySQL-8.0%2B-4479A1?logo=mysql)](https://www.mysql.com)

## Quick Navigation

| New to the project? | [Quick Start](#quick-start) |
|---------------------|---------------------------|
| Using the API? | [API Endpoints](#api-endpoints) |
| Contributing? | [Development](#development) |

## Table of Contents

- [Features](#features)
- [Core Concepts](#core-concepts)
- [Technology Stack](#technology-stack)
- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [API Endpoints](#api-endpoints)
- [Development](#development)

## Features

- **Flexible Consent Model**: Define consent elements (data points), group them into purposes, and track user approvals for any industry or solution
- **Complete Consent Lifecycle Management**: Create, retrieve, update, revoke, and validate user consents with full status tracking
- **Audit Trails**: Every status change is recorded for accountability and compliance
- **Multi-tenancy**: Organization-level data isolation via `org-id` header
- **Authorization Resources**: Track granular authorization status per user per consent
- **Attribute Search**: Query consents by custom metadata (key or key-value pairs)
- **Expiration Handling**: Automatic consent expiration with cascading status updates

## Core Concepts

OpenFGC is built on three core concepts:

```
┌───────────────────────────┐
│     Consent Elements      │  Data points or actions
│         (What)            │  e.g., user_email, location_tracking
└───────────────────────────┘
         │              ▲
     1:N │              │ 1:M
         ▼              │
┌───────────────────────────┐
│     Consent Purposes      │  Logical groupings of elements
│         (Why)             │  e.g., marketing, analytics
└───────────────────────────┘
         │              ▲
     1:N │              │ 1:M
         ▼              │
┌───────────────────────────┐
│        Consents           │  User approval record
│        (Record)           │  Links user → purposes → elements
└───────────────────────────┘
```

### 1. Consent Element
The Consent Element is the most granular unit of data or specific activity being consented to.

- **Definition**: The most granular unit—a specific data point (e.g., email address) or processing action (e.g., sharing with third parties).

### 2. Consent Purpose
The Consent Purpose provides the context and legal justification for the collection.

- **Definition**: A logical grouping of elements under a single objective. Instead of asking users about each data point, you present the reason for the request (e.g., "Marketing Communications" includes email and phone).

### 3. Consent (The Record)
The Consent is the immutable evidence of a user’s decision regarding specific Purposes.

- **Definition**: The record of a user's decision. Tracks who approved what, when, and maintains the full status lifecycle (Created → Active → Expired/Revoked) with audit trail.

## Technology Stack

- **Go** 1.25+
- **Web Framework**: net/http (standard library) with gorilla/mux style routing
- **Database**: MySQL 8.0+ or PostgreSQL 14+ (**recommended** for production; SQLite supported for development only)
- **ORM/Data Access**: sqlx
- **Architecture**: Domain-driven layered architecture
- **Transaction Management**: Atomic operations

## Prerequisites

- Go 1.25 or higher
- MySQL 8.0+ or PostgreSQL 14+ (**recommended** for production)
- sqlite3 (optional, if using SQLite)
- mysql (optional, if using MySQL integration tests)
- psql (optional, if using PostgreSQL integration tests)
- Make (optional, for build commands)

## Project Structure

```
openfgc/
├── api/                                    # OpenAPI specifications
│   ├── consent-management-API.yaml         # Consent API spec
├── consent-server/                         # Main application
│   ├── cmd/
│   │   └── server/
│   │       ├── main.go                     # Application entry point
│   │       └── servicemanager.go           # Service initialization
│   ├── internal/
│   │   ├── authresource/                   # Authorization resource module
│   │   ├── consent/                        # Consent module
│   │   ├── consentelement/                 # Consent element module
│   │   ├── consentpurpose/                 # Consent purpose module
│   │   └── system/                         # Shared system components
│   │       ├── config/                     # Configuration management
│   │       ├── database/                   # Database client & transactions
│   │       ├── error/                      # Error handling
│   │       ├── healthcheck/                # Health check endpoints
│   │       ├── log/                        # Logging infrastructure
│   │       ├── middleware/                 # HTTP middleware
│   │       ├── stores/                     # Store registry
│   │       └── utils/                      # Utilities
│   ├── dbscripts/
│   │   ├── db_schema_mysql.sql             # Consent tables schema (MySQL)
│   │   ├── db_schema_postgres.sql          # Consent tables schema (PostgreSQL)
│   │   ├── db_schema_sqlite.sql            # Consent tables schema (SQLite)
│   │   └── WIP-db_schema_config_mysql.sql  # Config tables schema
│   └── docs/                               # Internal documentation
├── tests/
│   └── integration/                        # Integration tests
│       ├── consent/                        # Consent API tests
│       ├── consentelement/                 # Consent element tests
│       ├── consentpurpose/                 # Consent purpose tests
│       └── main.go                         # Test runner
├── build.sh                                # Build script
├── start.sh                                # Server startup script
├── target/                                 # Build output directory (generated)
│   ├── server/                             # Runnable server artifacts
│   └── dist/                               # Distribution packages
└── version.txt                             # Version information
```

## Quick Start

### 1. Build

**Using build.sh (Recommended)**

```bash
# Build the application (binary only)
./build.sh build

# Create distribution package (binary + zip archive)
./build.sh package
```

Build artifacts are created in `target/server/`. Configs in `target/server/repository/conf/`

### 2. Setup Database

**SQLite:**

```bash
# Create the database directory
mkdir -p target/server/repository/database

# Initialize the SQLite database with the schema
sqlite3 target/server/repository/database/consent.db < consent-server/dbscripts/db_schema_sqlite.sql
```

**MySQL:**

```bash
# Create database
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS consent_mgt;"

# Import schema
mysql -u root -p consent_mgt < consent-server/dbscripts/db_schema_mysql.sql
```

**PostgreSQL:**

```bash
# Create database
psql -U postgres -c "CREATE DATABASE consent_mgt;"

# Import schema
psql -U postgres -d consent_mgt -f consent-server/dbscripts/db_schema_postgres.sql
```

### 3. Configure Application

The default configuration uses SQLite. Update configuration file at `target/server/repository/conf/deployment.yaml`:

```yaml
server:
  hostname: 0.0.0.0
  port: 8060
  readTimeout: 30s
  writeTimeout: 30s
  idleTimeout: 120s

database:
  consent:
    type: sqlite
    path: ${OPENFGC_DB_PATH}  #e.g. ./repository/database/consent.db
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    options: ""               # e.g. _pragma=journal_mode(WAL)&_pragma=cache_size(-16000)

logging:
  level: info

consent:
  periodical_expiration:
    enabled: false
    frequency: "1h"
    eligible_statuses: ["ACTIVE"]
  status_mappings:
    active_status: ACTIVE
    expired_status: EXPIRED
    revoked_status: REVOKED
    created_status: CREATED
    rejected_status: REJECTED
  auth_status_mappings:
    approved_state: APPROVED
    rejected_state: REJECTED
    created_state: CREATED
    system_expired_state: SYS_EXPIRED
    system_revoked_state: SYS_REVOKED
  history:
    enabled: false
```

For MySQL, set `type: mysql` and set the following database parameters:

```yaml
database:
  consent:
    type: ${OPENFGC_DB_TYPE}
    hostname: ${OPENFGC_DB_HOSTNAME}
    port: ${OPENFGC_DB_PORT}
    database: ${OPENFGC_DB_NAME}
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    user: ${OPENFGC_DB_USER}
    password: ${OPENFGC_DB_PASSWORD}
```

For PostgreSQL, set `type: postgres` and set the following database parameters:

```yaml
database:
  consent:
    type: ${OPENFGC_DB_TYPE}
    hostname: ${OPENFGC_DB_HOSTNAME}
    port: ${OPENFGC_DB_PORT}
    database: ${OPENFGC_DB_NAME}
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    user: ${OPENFGC_DB_USER}
    password: ${OPENFGC_DB_PASSWORD}
    sslmode: disable        # use verify-full for production
    options: ""             # e.g. sslrootcert=/path/to/ca.crt for production TLS
```

Configuration values are read from `deployment.yaml`. Environment variables are substitued where the file contains `${VARIABLE_NAME}` placeholders. You can either replace those placeholders with literal values or set the variables before starting the server.

| Variable | Description | Example Values |
|----------|-------------|----------------|
| `OPENFGC_DB_TYPE` | Database type | `mysql`, `sqlite`, `postgres` |
| `OPENFGC_DB_HOSTNAME` | Database hostname for MySQL/PostgreSQL | `localhost` |
| `OPENFGC_DB_PORT` | Database port for MySQL/PostgreSQL | `3306`, `5432` |
| `OPENFGC_DB_NAME` | Database name for MySQL/PostgreSQL | `consent_mgt` |
| `OPENFGC_DB_USER` | Database user for MySQL/PostgreSQL | `root`, `postgres` |
| `OPENFGC_DB_PASSWORD` | Database password for MySQL/PostgreSQL | `password` |
| `OPENFGC_DB_PATH` | SQLite database file path | `./repository/database/consent.db` |
| `OPENFGC_DB_SSLMODE` | PostgreSQL SSL mode | `disable`, `verify-full` |
| `OPENFGC_DB_OPTIONS` | Optional DB-specific connection options | `_pragma=journal_mode(WAL)&_pragma=cache_size(-16000)` |

### 4. Run

```bash
# Run in normal mode
cd target/server
./start.sh

# Run in debug mode (with remote debugging on port 2345)
./start.sh --debug

# Run in debug mode with custom port
./start.sh --debug --debug-port 3456
```

Server starts at `http://localhost:8060`

Health check: `curl http://localhost:8060/health`

## API Endpoints

- [Open Fine-Grained Consent API schema](api/consent-management-API.yaml)

> **Tip:** You can import these OpenAPI specifications directly into [Postman](https://www.postman.com/) or similar tools to easily explore and test the API.

All requests require headers:
- `org-id`: Organization identifier

## Development

### Build from Source

```bash
# Navigate to server directory
cd consent-server

# Build binary
go build -o bin/consent-server cmd/server/main.go

# Run
./bin/consent-server
```

### Run Tests

**Using build.sh (Recommended)**

```bash
# Run unit tests
./build.sh test_unit

# Run integration tests (MySQL by default)
./build.sh test_integration

# Run integration tests against SQLite or PostgreSQL 
DB_TYPE=sqlite ./build.sh test_integration
DB_TYPE=postgres ./build.sh test_integration

# Run all tests
./build.sh test
```

> **Note:** Integration tests obtain their configuration from `tests/integration/repository/conf/` based on `DB_TYPE`. If you're using a separate database for testing, update the matching configuration file before running the suite. The test database will be automatically recreated and initialized with the required schema.

**Manual Execution**

```bash
# Navigate to test directory
cd tests/integration

# Run all tests
go test ./... -v
```