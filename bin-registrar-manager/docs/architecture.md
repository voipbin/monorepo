# bin-registrar-manager — Architecture

## Component Overview

`bin-registrar-manager` is a Class A Go RPC microservice that manages SIP registrations for the VoIPbin platform. It handles the lifecycle of Asterisk PJSIP extensions (user endpoints) and trunks (carrier connections), including provisioning of Asterisk database tables and caching of active contact registrations.

This service is unique in using **two separate databases**: the Asterisk DB (`ps_*` tables) and the bin-manager DB (extensions, trunks, sip_auth).

**Binary:** `registrar-manager` (daemon) + `registrar-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/registrar-manager` | Daemon entry point; establishes both DB connections |
| `cmd/registrar-control` | CLI tool for direct management (bypasses RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/subscribehandler` | Event subscriber for cascading customer deletes |
| `pkg/extensionhandler` | Business logic for SIP extension lifecycle |
| `pkg/trunkhandler` | Business logic for SIP trunk lifecycle |
| `pkg/contacthandler` | Reads active SIP contacts from Asterisk DB; Redis-cached |
| `pkg/dbhandler` | Unified DB abstraction over both MySQL databases |
| `pkg/cachehandler` | Redis-backed contact cache |

## Layer Responsibilities

```
RabbitMQ
   │
   ├── listenhandler       ← RPC requests (extensions, trunks, contacts)
   │       │
   │       ├── extensionhandler  ← SIP extension CRUD
   │       │       │
   │       │       └── dbhandler ← bin-manager DB (extensions table)
   │       │                       + Asterisk DB (ps_endpoints, ps_aors, ps_auths)
   │       │
   │       ├── trunkhandler      ← SIP trunk CRUD
   │       │       │
   │       │       └── dbhandler ← bin-manager DB (trunks table)
   │       │                       + Asterisk DB (ps_endpoints, ps_aors, ps_auths)
   │       │
   │       └── contacthandler    ← Active SIP registrations (read-only)
   │               │
   │               └── dbhandler + cachehandler ← Asterisk ps_contacts + Redis
   │
   └── subscribehandler    ← customer_deleted → cleanup extensions + trunks
```

- **listenhandler**: URI regex routing; no business logic.
- **extensionhandler**: Creates/deletes extension records in bin-manager DB AND corresponding `ps_endpoints`/`ps_aors`/`ps_auths` entries in Asterisk DB atomically.
- **trunkhandler**: Same pattern for trunks; supports basic (user/pass) and IP-based authentication modes.
- **contacthandler**: Read-only view of active registrations from Asterisk `ps_contacts` table; Redis-cached.
- **dbhandler**: Abstracts both DB connections; uses `Masterminds/squirrel` for query building.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.registrar-manager.request`. The `listenhandler` matches the URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/contacts(\?.*)?$` | GET | List active SIP contacts (from Asterisk ps_contacts) |
| `/v1/extensions/count_by_customer$` | GET | Count extensions per customer |
| `/v1/extensions$` | POST | Create SIP extension |
| `/v1/extensions\?` | GET | List extensions with filters |
| `/v1/extensions/{uuid}/direct-hash-regenerate$` | POST | Regenerate extension's direct hash |
| `/v1/extensions/{uuid}$` | GET, PUT, DELETE | Get / update / delete extension |
| `/v1/extensions/endpoint/{uuid}$` | GET | Get extension by Asterisk endpoint ID |
| `/v1/extensions/extension/{uuid}(\?.*)?$` | GET | Get extension by extension number |
| `/v1/trunks/count_by_customer$` | GET | Count trunks per customer |
| `/v1/trunks$` | POST | Create SIP trunk |
| `/v1/trunks\?` | GET | List trunks with filters |
| `/v1/trunks/{uuid}$` | GET, PUT, DELETE | Get / update / delete trunk |
| `/v1/trunks/domain_name/` | GET | Get trunk by domain name |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.
