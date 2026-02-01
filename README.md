# Consent Management API

A high-performance RESTful API service for managing user consents, consent purposes, and authorization resources with organization-level multi-tenancy support.

## Features

- **Consent Management**: Create, retrieve, update, revoke, and validate consents
- **Consent Purposes**: Define and manage consent purposes with type-based validation
- **Authorization Resources**: Handle consent authorization resources with status tracking
- **Attribute Search**: Search consents by custom attributes (key or key-value pairs)
- **Status Auditing**: Complete audit trail for consent status changes
- **Multi-tenancy**: Organization-level data isolation with `org-id` header
- **Expiration Handling**: Automatic consent expiration with cascading status updates

## Technology Stack

- **Go** 1.21+
- **Web Framework**: net/http (standard library)
- **Database**: MySQL 8.0+
- **Architecture**: Clean architecture with layered design
- **Transaction Management**: atomic operations

## Prerequisites

- Go 1.21 or higher
- MySQL 8.0 or higher

## Project Structure

```
consent-mgt-v1/
├── api/                                    # OpenAPI specifications
│   ├── consent-management-API.yaml        # Consent API spec
│   └── config-management-API.yaml         # Config API spec
├── consent-server/                         # Main application
│   ├── cmd/
│   │   └── server/
│   │       ├── main.go                    # Application entry point
│   │       └── servicemanager.go          # Service initialization
│   ├── internal/
│   │   ├── consent/                       # Consent module
│   │   │   ├── handler.go                # HTTP handlers
│   │   │   ├── service.go                # Business logic
│   │   │   ├── store.go                  # Data access layer
│   │   │   ├── init.go                   # Route registration
│   │   │   ├── model/                    # Domain models
│   │   │   └── validator/                # Request validators
│   │   ├── consentpurpose/               # Consent purpose module
│   │   ├── authresource/                 # Auth resource module
│   │   └── system/                       # Shared system components
│   │       ├── config/                   # Configuration management
│   │       ├── database/                 # Database client & transactions
│   │       ├── error/                    # Error handling
│   │       ├── middleware/               # HTTP middleware
│   │       ├── stores/                   # Store registry
│   │       └── utils/                    # Utilities
│   ├── dbscripts/
│   │   ├── db_schema_mysql.sql           # Consent tables schema
│   │   └── db_schema_config_mysql.sql    # Config tables schema
│   └── bin/                              # Build output directory
├── tests/integration/                     # Integration tests
│   └── api/
│       ├── consent/                      # Consent API tests
│       ├── consent-purpose/              # Purpose API tests
│       └── auth_resource_api_test.go     # Auth resource tests
├── build.sh                              # Build script
├── start.sh                              # Server startup script
└── version.txt                           # Version information
```

## Quick Start

### 1. Setup Database

```bash
# Create database
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS consent_mgt;"

# Import schemas
mysql -u root -p consent_mgt < consent-server/dbscripts/db_schema_mysql.sql
```

### 2. Build

**Using build.sh (Recommended)**

```bash
# Build the application
./build.sh build

# This creates:
# - consent-server/bin/consent-server (binary)
# - consent-server/bin/repository/conf/ (config directory)
# - consent-server/bin/api/ (API specs)
# - consent-server/bin/dbscripts/ (database scripts)
```

### 3. Configure Application

Update configuration file at `consent-server/bin/repository/conf/deployment.yaml`:

```yaml
server:
  port: 3000
  host: "0.0.0.0"

database:
  host: "localhost"
  port: 3306
  username: "root"
  password: "your_password"
  database: "consent_mgt"
  max_open_connections: 25
  max_idle_connections: 10
  connection_max_lifetime_minutes: 5
```


### 2. Run

```bash
# Run in normal mode
cd bin
./start.sh

# Run in debug mode (with remote debugging on port 2345)
./start.sh --debug

# Run in debug mode with custom port
./start.sh --debug --debug-port 3000
```

Server starts at `http://localhost:3000`

Health check: `curl http://localhost:3000/health`

## API Endpoints

[API schema](api/consent-management-API.yaml)

All requests require headers:
- `org-id`: Organization identifier
- `client-id`: Client application identifier

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

```bash
# Navigate to test directory
cd tests/integration

# Run all tests
go test ./... -v

# Run specific module tests
go test ./api/consent/... -v
go test ./api/consent-purpose/... -v

# Run with coverage
go test ./... -v -cover
```