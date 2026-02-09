
# OpenFGC Architecture Documentation

This document provides an overview of the OpenFGC (Open Fine-Grained Consent) architecture for developers.

## Architecture Overview

OpenFGC follows a **layered architecture** with clear separation of concerns:

```text
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Layer                              │
│                   (Handlers - Routing & Validation)             │
│                                                                 │
│    Consent  │  Purpose  │  Element  │  Auth Resource           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Service Layer                               │
│              (Business Logic & Orchestration)                    │
│                                                                 │
│    Consent  │  Purpose  │  Element  │  Auth Resource           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                       Store Layer                                │
│                      (Data Access)                               │
│                                                                 │
│    Consent  │  Purpose  │  Element  │  Auth Resource           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                       Database Layer                             │
│                         MySQL 8.0+                               │
│                                                                 │
│  CONSENT  │  CONSENT_PURPOSE  │  CONSENT_ELEMENT  │  AUTH_RESOURCE │
└─────────────────────────────────────────────────────────────────┘

Cross-Cutting: Middleware, Config, Health Check, Error Handling
```

## Layer Responsibilities

### 1. Handler Layer
**What it does**: Handles incoming HTTP requests and returns responses

- Receives API calls from clients
- Extracts request data (headers, parameters, body)
- Calls the appropriate service method
- Formats responses and error messages
- Sets HTTP status codes

### 2. Service Layer
**What it does**: Implements business rules and coordinates operations

- Validates business logic constraints
- Coordinates multiple database operations
- Manages database transactions
- Handles status calculations and transitions
- Returns structured errors

### 3. Store Layer
**What it does**: Interacts directly with the database

- Executes SQL queries or provide queries.
- Maps database results to application objects
- Handles filtering and pagination
- No business logic - pure database access

## Domain Models

The system manages four core entities:

### 1. Consent Element
**What it is**: The smallest unit of consent - a specific piece of data or action

**Purpose**: Represents individual items that users can consent to, such as:
- Personal data fields (email, phone, address)
- Processing activities (marketing, analytics)
- Access permissions (account access, transaction history)

**Types**:
- **basic**: Simple consent items without additional structure
- **json-payload**: Elements with structured data (requires validation schema)
- **resource-field**: Elements tied to specific resource paths (requires resource path and JSON path)

**Database**: `CONSENT_ELEMENT` table

### 2. Consent Purpose
**What it is**: A logical grouping of consent elements with a specific context

**Purpose**: Groups related elements together under a meaningful purpose, such as:
- "Marketing Communications" (email + phone + name)
- "Account Management" (profile access + transaction history)
- "Analytics" (usage data + location)

**Key Features**:
- Links multiple elements together
- Each element can be marked as mandatory or optional
- Scoped to specific clients within an organization

**Database**: `CONSENT_PURPOSE` + `PURPOSE_ELEMENT_MAPPING` tables

### 3. Consent
**What it is**: The authoritative record of a user's consent decision

**Purpose**: Tracks the user's agreement or rejection of specific purposes

**Status Lifecycle**:
- **CREATED**: Consent created, awaiting authorization
- **ACTIVE**: User has authorized the consent
- **REJECTED**: User has rejected the consent
- **REVOKED**: User has withdrawn a previously given consent
- **EXPIRED**: Active consent has passed its validity period

**Key Features**:
- Links to one or more purposes
- Tracks user approvals/rejections for each element
- Manages validity periods and expiration
- Supports recurring consent patterns

**Database**: `CONSENT` + `CONSENT_PURPOSE_MAPPING` + `CONSENT_ELEMENT_APPROVAL` tables

### 4. Authorization Resource
**What it is**: Represents a user's authorization decision for consent

**Purpose**: Captures the actual user authorization event and associated resources

**Key Features**:
- Links to a specific consent
- Tracks authorization status (CREATED, APPROVED, REJECTED, REVOKED)
- Stores authorized resources (accounts, transactions, etc.) as flexible JSON
- Connects user identity to consent

**Status Impact**: The authorization status drives the consent status:
- Authorization APPROVED → Consent becomes ACTIVE
- Authorization REJECTED → Consent becomes REJECTED
- Authorization REVOKED → Consent becomes REVOKED

**Database**: `AUTH_RESOURCE` table

## Key Design Principles

### 1. Status Derivation
**Why**: Consent status is automatically synchronized with authorization decisions

The system automatically updates consent status based on authorization events:
- When user authorizes → Consent becomes ACTIVE
- When user rejects → Consent becomes REJECTED  
- When user revokes → Consent becomes REVOKED
- When validity expires (only for ACTIVE) → Consent becomes EXPIRED

**Important**: Terminal statuses (REJECTED, REVOKED) never change to EXPIRED, even if validity passes.

### 2. Type-Specific Validation
**Why**: Different element types have different requirements

- **basic**: No special requirements
- **json-payload**: Requires a validation schema to validate structured data
- **resource-field**: Requires resource path and JSON path to locate data in external resources

### 3. Multi-Tenancy
**Why**: Isolate data between different organizations and clients

All entities are scoped by organization ID. Additionally, purposes are scoped by client ID to allow different clients within the same organization to define their own consent purposes.