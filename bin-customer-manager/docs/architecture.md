# bin-customer-manager — Architecture

## Component Overview

`bin-customer-manager` is a Class A Standard Go RPC Manager and the **foundational identity service** for the VoIPbin platform. It manages tenant organizations (customers) and their API credentials (access keys). Almost every other service depends on customer context for authorization and isolation — this service is the root of the tenant hierarchy.

```
cmd/customer-manager/main.go
    ├── pkg/cachehandler      (Redis — cache-first reads)
    ├── pkg/dbhandler         (MySQL — parameterized raw SQL)
    ├── pkg/customerhandler   (Customer CRUD, cross-service validation, email verification)
    ├── pkg/accesskeyhandler  (Access key CRUD)
    └── pkg/listenhandler     (RabbitMQ RPC — customers & accesskeys API)
```

Supporting binary:
- `cmd/customer-control/` — CLI tool for direct DB/cache operations, bypasses RabbitMQ RPC.

**Foundational role:** `bin-customer-manager` has no SubscribeHandler — it does not consume events from other services. All state changes are driven by inbound RPC requests. Because every other service scopes resources by `customer_id`, this service must be healthy for the platform to function.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/customer-manager` | Cobra + Viper config, dependency wiring, daemon start |
| Listen | `pkg/listenhandler` | RabbitMQ RPC routing; dispatches to customerhandler / accesskeyhandler |
| Business | `pkg/customerhandler` | Customer lifecycle, email verification, freeze/recover, cross-service validation |
| Business | `pkg/accesskeyhandler` | Access key creation, rotation, deletion |
| Data | `pkg/dbhandler` | Parameterized raw SQL queries (no query builder) |
| Cache | `pkg/cachehandler` | Redis cache-first reads; invalidated on mutations |
| Models | `models/customer` | Customer, WebhookMessage, validation |
| Models | `models/accesskey` | AccessKey, WebhookMessage |

## Request Routing

The `listenhandler` consumes from queue `bin-manager.customer-manager.request` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| POST | `/v1/customers` | Create customer |
| GET | `/v1/customers?` | List customers (cursor pagination) |
| GET | `/v1/customers/{uuid}` | Get customer |
| PUT | `/v1/customers/{uuid}` | Update customer |
| DELETE | `/v1/customers/{uuid}` | Delete customer |
| PUT | `/v1/customers/{uuid}/billing_account_id` | Link billing account |
| PUT | `/v1/customers/{uuid}/metadata` | Update metadata |
| POST | `/v1/customers/signup` | New customer self-registration |
| POST | `/v1/customers/email_verify` | Email verification callback |
| POST | `/v1/customers/{uuid}/freeze` | Freeze customer account |
| POST | `/v1/customers/{uuid}/recover` | Recover frozen customer |
| POST | `/v1/customers/{uuid}/freeze_and_delete` | Freeze and schedule deletion |
| POST | `/v1/accesskeys` | Create access key |
| GET | `/v1/accesskeys?` | List access keys |
| GET | `/v1/accesskeys/{uuid}` | Get access key |
| PUT | `/v1/accesskeys/{uuid}` | Update access key |
| DELETE | `/v1/accesskeys/{uuid}` | Delete access key |
