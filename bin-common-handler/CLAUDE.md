# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`bin-common-handler` is a shared library within a Go monorepo providing common functionality for VoIP services. It contains reusable handlers, models, and utilities used across multiple manager services (bin-agent-manager, bin-call-manager, bin-billing-manager, etc.).

## Commands

### Testing
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./pkg/utilhandler

# Run a specific test
go test ./pkg/utilhandler -run TestIsValidEmail

# Run tests with verbose output
go test -v ./...
```

### Code Generation
```bash
# Generate mocks for all packages (from project root)
go generate ./...

# Generate mocks for a specific package
go generate ./pkg/notifyhandler
```

### Dependency Management
```bash
# Update dependencies
go get -u ./...

# Tidy and vendor dependencies
go mod tidy
go mod vendor
```

### Update All Projects in Monorepo
From the monorepo root:
```bash
ls -d */ | xargs -I {} bash -c "cd '{}' && go get -u ./... && go mod vendor && go generate ./... && go test ./..."
```

## Architecture

### Monorepo Structure
This is part of a Go monorepo with local module replacements. All `monorepo/bin-*` dependencies are resolved to sibling directories via `replace` directives in go.mod.

### Package Organization

**models/** - Core data structures and types
- `identity/` - Resource identity types (ID, CustomerID)
- `sock/` - Message broker types (Request, Response, Event)
- `address/` - Address-related models
- `service/` - Service definitions
- `outline/` - Service names and queue names

**pkg/** - Reusable handler packages
- `requesthandler/` - Inter-service RPC request handlers for all manager services
- `notifyhandler/` - Event publishing and webhook notification handlers
- `rabbitmqhandler/` - RabbitMQ connection and queue management
- `sockhandler/` - Abstract message broker interface (currently RabbitMQ)
- `databasehandler/` - Database utilities and helpers
- `utilhandler/` - General utilities (UUID, hashing, time, email validation, URL)

### Key Architectural Patterns

**Inter-Service Communication**
Services communicate via RabbitMQ using a request/response pattern:
- `sock.Request` - RPC-style requests with URI, Method (GET/POST/PUT/DELETE), Publisher, and Data
- `sock.Response` - Responses with StatusCode, DataType, and Data
- `sock.Event` - Pub/sub events with Type, Publisher, and Data

**RequestHandler Pattern**
The `requesthandler` package contains typed methods for making RPC requests to all manager services:
- Each manager service has dedicated methods (e.g., `BillingV1BillingGets`, `CallV1CallCreate`)
- Methods construct sock.Request messages and send them via RabbitMQ
- Responses are unmarshaled into typed structs from the target service's models package

**NotifyHandler Pattern**
- `PublishEvent` - Publish events to topic exchanges
- `PublishWebhook` - Send webhook notifications to customer endpoints
- Supports delayed message delivery via RabbitMQ delay exchange

**Mock Generation**
All main handler interfaces use go:generate with mockgen:
```go
//go:generate mockgen -package packagename -destination ./mock_main.go -source main.go -build_flags=-mod=mod
```

### Queue Types
- **Normal queues** - Durable, survive broker restarts
- **Volatile queues** - Auto-delete when unused, for temporary operations

### Identity Model
All resources follow a common identity pattern with:
- `ID` - Resource UUID
- `CustomerID` - Owning customer UUID

## Testing Patterns

- Test files use `_test.go` suffix
- Table-driven tests are standard
- Mocks are generated via mockgen and stored as `mock_*.go`
- Tests verify type conversions (UUID to bytes, JSON marshaling)
