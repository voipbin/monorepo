# Developer Operations Toolkit Design

**Date:** 2026-04-10
**Status:** Approved
**Author:** AI-assisted

## Problem

Developers working in the voipbin monorepo face significant daily friction:

1. **Manual 5-step verification** must be run before every commit:
   `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
   This takes 5-10 minutes per run and is easy to forget steps.

2. **The `go clean -testcache` footgun**: After any `bin-common-handler` change, cached test results mask failures. This is documented as the #1 cause of CI failures.

3. **Cascading common-handler updates** require running the full verification across all 30+ services manually, with no automation or parallelism.

4. **No local dev environment**: No docker-compose file to spin up MySQL, Redis, and RabbitMQ for local development or testing.

5. **No prerequisites checker**: Developers must manually verify they have the right Go version, golangci-lint, and other tools installed.

## Solution

A developer operations toolkit consisting of:

- **Root `Makefile`**: Unified entry point for all common operations
- **`scripts/verify.sh`**: Automated 5-step verification with proper cache clearing
- **`scripts/update-common.sh`**: Cascading bin-common-handler updates across all services
- **`docker-compose.dev.yml`**: Local development infrastructure
- **`scripts/doctor.sh`**: Environment prerequisites checker

## Components

### Makefile

Provides these targets:
- `make verify SERVICE=<name>` - Full 5-step verification for one service
- `make verify-all` - Verify all Go services in parallel
- `make verify-common` - Special workflow for common-handler changes
- `make test SERVICE=<name>` - Run tests with cache clearing
- `make lint SERVICE=<name>` - Run linting only
- `make generate SERVICE=<name>` - Run code generation only
- `make build SERVICE=<name>` - Build the service binary
- `make dev-up` / `make dev-down` - Start/stop local infrastructure
- `make doctor` - Check prerequisites
- `make services` - List all services

### scripts/verify.sh

Core verification engine that:
- Runs all 5 steps in correct order
- Always clears test cache (prevents the #1 footgun)
- Supports single service or all-services mode
- Reports timing and pass/fail per step
- Returns non-zero exit code on any failure

### scripts/update-common.sh

Common handler cascade updater that:
- First verifies bin-common-handler itself
- Then runs full verification across all dependent services
- Supports parallel execution with configurable concurrency
- Produces a summary report of pass/fail per service

### docker-compose.dev.yml

Local dev infrastructure:
- MySQL 8.0 on port 3306 (matching service config defaults)
- Redis 7 on port 6379 with password
- RabbitMQ 3.12 on ports 5672 (AMQP) and 15672 (management UI)

### scripts/doctor.sh

Prerequisites checker that verifies:
- Go version (1.25+)
- golangci-lint installed
- Docker and Docker Compose available
- mockgen installed
- Required CLI tools present

## Design Decisions

1. **Shell scripts over Go CLI**: The toolkit is shell-based because it orchestrates existing Go tooling. A Go CLI would add a build step and dependency management overhead for simple orchestration.

2. **Makefile as entry point**: Make is universally available, self-documenting (via `make help`), and familiar to Go developers.

3. **Parallel execution**: `update-common.sh` uses `xargs -P` for parallel service verification, significantly reducing wall-clock time.

4. **Docker Compose for dev**: Provides consistent, reproducible local infrastructure without requiring manual setup of MySQL, Redis, and RabbitMQ.

## File Layout

```
monorepo/
├── Makefile
├── docker-compose.dev.yml
└── scripts/
    ├── verify.sh
    ├── update-common.sh
    └── doctor.sh
```

## Success Criteria

- `make doctor` validates development environment
- `make verify SERVICE=bin-agent-manager` completes the full 5-step verification automatically
- `make verify-common` detects failures across all services after a common-handler change
- `make dev-up` starts MySQL, Redis, RabbitMQ for local development
- All scripts have `--help` flags and clear error messages
