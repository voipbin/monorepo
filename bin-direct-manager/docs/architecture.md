# bin-direct-manager — Architecture

## Component Overview

`bin-direct-manager` is a Class A Go RPC microservice that manages direct hash records for SIP URI routing. Each direct record maps a unique, regeneratable hash to a customer resource (extension, conference, AI agent, AI team, or human agent), enabling direct SIP URI dialing without requiring a phone number.

**Binary:** `direct-manager` (daemon) + `direct-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/direct-manager` | Daemon entry point; wires config, DB, cache, and handlers |
| `cmd/direct-control` | CLI tool for direct DB/cache management (bypasses RabbitMQ) |
| `pkg/config` | Configuration singleton via Cobra + Viper |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/subscribehandler` | Event subscriber for customer deletion cascades |
| `pkg/directhandler` | Core business logic for direct hash CRUD and regeneration |
| `pkg/dbhandler` | MySQL operations via `Masterminds/squirrel` |
| `pkg/cachehandler` | Redis cache for hash-based lookups |
| `models/direct` | Direct struct, event types, webhook |

## Layer Responsibilities

```
RabbitMQ
   │
   ├── listenhandler      ← RPC requests (CRUD, hash lookup, regenerate)
   │       │
   │       └── directhandler   ← business logic, hash generation
   │               │
   │               ├── dbhandler      ← MySQL (direct records)
   │               └── cachehandler   ← Redis (hash index)
   │
   └── subscribehandler   ← customer_deleted events → cascade delete
```

- **listenhandler**: URI regex routing only; no business logic.
- **directhandler**: Manages direct hash lifecycle including random hash generation and regeneration. Calls dbhandler and cachehandler.
- **dbhandler**: `Masterminds/squirrel` SQL builder; soft-delete pattern.
- **cachehandler**: Redis hash index for O(1) lookup by hash value.
- **subscribehandler**: Handles `customer_deleted` events by removing all direct records for that customer.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.direct-manager.request`. The `listenhandler` matches the URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/directs$` | POST | Create a new direct hash |
| `/v1/directs\?` | GET | List directs (pagination via page_size/page_token) |
| `/v1/directs/by-hash/` | GET | Get direct by hash value |
| `/v1/directs/{uuid}/regenerate$` | POST | Regenerate hash for existing direct |
| `/v1/directs/{uuid}$` | GET, DELETE | Get or delete direct by ID |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.
