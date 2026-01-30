# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-storage-manager is a Go microservice for managing file and media storage in a VoIP system. It handles file uploads, downloads, recordings, and compression operations using Google Cloud Storage (GCS). The service manages customer storage accounts with quota enforcement (10GB per customer) and integrates with other microservices via RabbitMQ.

## Build and Test Commands

```bash
# Build the daemon
go build -o ./bin/storage-manager ./cmd/storage-manager

# Run the daemon (requires configuration via flags or env vars)
./bin/storage-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Run a specific package's tests
go test -v ./pkg/filehandler/...

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/filehandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/accounthandler/...
go generate ./pkg/storagehandler/...
go generate ./pkg/subscribehandler/...

# Vet
go vet $(go list ./...)
```

## storage-control CLI Tool

A command-line tool for managing storage files and accounts directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# File commands
./bin/storage-control file create --customer_id <uuid> --name <name> [--detail] [--reference_type] [--reference_id]
./bin/storage-control file get --id <uuid>
./bin/storage-control file list --customer_id <uuid> [--limit 100] [--token]
./bin/storage-control file delete --id <uuid>

# Account commands
./bin/storage-control account create --customer_id <uuid> [--name] [--detail]
./bin/storage-control account get --id <uuid>
./bin/storage-control account list --customer_id <uuid> [--limit 100] [--token]
./bin/storage-control account delete --id <uuid>

# Recording commands
./bin/storage-control recording get --id <uuid>
./bin/storage-control recording list --customer_id <uuid> [--limit 100] [--token]
./bin/storage-control recording delete --id <uuid>
```

Uses same environment variables as storage-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/storage-manager/** - Main daemon entry point with configuration via Viper/pflag (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like API operations (`/v1/files`, `/v1/accounts`, `/v1/recordings`, `/v1/compressfiles`)
3. **pkg/subscribehandler/** - Event subscriber handling cascading deletions from customer-manager service
4. **pkg/storagehandler/** - Core business logic layer coordinating file, recording, and compression operations
5. **pkg/filehandler/** - GCS bucket operations, file lifecycle management, and signed URL generation
6. **pkg/accounthandler/** - Storage account management with quota enforcement (10GB limit per customer)
7. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
8. **pkg/cachehandler/** - Redis cache operations for file and account lookups
9. **models/** - Data structures (file, account, bucketfile, compressfile)

### GCS Authentication Patterns

The filehandler supports two authentication modes:
1. **Service Account JSON**: When `GOOGLE_APPLICATION_CREDENTIALS` env var points to a JSON file, private keys are extracted for local signing
2. **ADC/Workload Identity**: In GCE/GKE environments, uses Application Default Credentials with IAM Credentials API for signing

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Publishes events to `QueueNameStorageEvent` when files/accounts change
- Subscribes to `QueueNameCustomerEvent` for cascading deletions when customers are deleted
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package
- Prometheus metrics exposed at configurable endpoint
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`
- Joonix logging format for structured logs (GCP-compatible)

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → storagehandler → filehandler → GCS bucket
                                                                  ↓
                                                              accounthandler (quota check)
                                                                  ↓
                                                              dbhandler → MySQL/Redis
```

### Storage Architecture

1. **Two GCS Buckets**:
   - `gcp_bucket_name_media` - Persistent storage for recordings, files
   - `gcp_bucket_name_tmp` - Temporary storage for compressed files (zip archives)

2. **Bucket Directory Structure**:
   - `recording/` - Call recordings
   - `bin/` - Service-uploaded files
   - `tmp/` - Temporary compressed files with SHA-1 hash naming

3. **File Reference Types**:
   - `normal` - Generic uploaded files
   - `recording` - Call recordings (can be compressed into zip for download)

4. **Download URL Generation**:
   - Uses GCS signed URLs with configurable expiration (default 24 hours)
   - Supports local signing (with private key) or IAM Credentials API

### API Endpoints (via RabbitMQ RPC)

The service exposes REST-like endpoints through RabbitMQ:

**Accounts**:
- `POST /v1/accounts` - Create storage account
- `GET /v1/accounts?page_size=X&page_token=Y` - List accounts
- `GET /v1/accounts/<id>` - Get account details
- `DELETE /v1/accounts/<id>` - Delete account

**Files**:
- `POST /v1/files` - Create file record
- `GET /v1/files?page_size=X&page_token=Y` - List files
- `GET /v1/files/<id>` - Get file with download URL
- `DELETE /v1/files/<id>` - Delete file

**Recordings**:
- `GET /v1/recordings/<reference_id>` - Get compressed recording (creates zip of all files with reference_id)
- `DELETE /v1/recordings/<reference_id>` - Delete all files for a recording

**Compress Files**:
- `POST /v1/compressfiles` - Create compressed archive from multiple files

### Configuration

Environment variables / flags (via Viper binding):
- `DATABASE_DSN` - MySQL connection string (format: `user:password@tcp(host:port)/dbname`)
- `RABBITMQ_ADDRESS` - RabbitMQ connection (format: `amqp://user:pass@host:port`)
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache configuration
- `GCP_PROJECT_ID` - Google Cloud project ID
- `GCP_BUCKET_NAME_MEDIA` - GCS bucket for persistent media storage
- `GCP_BUCKET_NAME_TMP` - GCS bucket for temporary files
- `GOOGLE_APPLICATION_CREDENTIALS` - (optional) Path to service account JSON for local GCS authentication
- `GOOGLE_SERVICE_ACCOUNT_EMAIL` - (optional) Service account email when not running on GCE
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint configuration

### Important Constraints

- **Storage Quota**: Each customer account has a 10GB storage limit (enforced in accounthandler:17)
- **Recording Compression**: Recordings with the same reference_id are automatically compressed into a single zip file on download
- **Cascading Deletions**: When a customer is deleted, all associated accounts and files are automatically removed
- **Cache Coordination**: File and account operations update both MySQL and Redis for consistency
