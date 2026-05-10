# bin-contact-manager — Architecture

## Component Overview

`bin-contact-manager` is a Class A Go RPC microservice that manages CRM-style contact records for the VoIPbin platform. It stores contacts belonging to customers, with support for multiple phone numbers, emails, and tag assignments. It also serves O(1) lookup requests used by inbound call flows for caller-ID enrichment.

**Binary:** `contact-manager` (daemon) + `contact-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/contact-manager` | Daemon entry point; wires config, DB, cache, and handlers |
| `cmd/contact-control` | CLI tool for direct DB/cache management (bypasses RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/subscribehandler` | Event subscriber for cascading deletes from customer-manager |
| `pkg/contacthandler` | Core business logic for contacts, phone numbers, emails, tags |
| `pkg/dbhandler` | MySQL operations; coordinates with Redis for cache invalidation |
| `pkg/cachehandler` | Redis-backed lookup cache for contacts by phone/email |
| `models/contact` | Contact, PhoneNumber, Email structs, event types, webhook |

## Layer Responsibilities

```
RabbitMQ
   │
   ├── listenhandler      ← RPC requests (CRUD, lookup)
   │       │
   │       └── contacthandler  ← business logic, validation
   │               │
   │               ├── dbhandler      ← MySQL (contacts, phone-numbers, emails, tags)
   │               ├── cachehandler   ← Redis (lookup index by phone/email)
   │               └── notifyhandler  ← publishes contact_created/updated/deleted events
   │
   └── subscribehandler   ← customer_deleted events → cascade delete
```

- **listenhandler**: Parses request URI/method, dispatches to contacthandler. No business logic.
- **contacthandler**: Owns all CRUD operations, tag linking, and event publishing. Calls dbhandler and cachehandler.
- **dbhandler**: Wraps MySQL with `Masterminds/squirrel`. Owns soft-delete (`tm_delete`) lifecycle.
- **cachehandler**: Redis hash-based index allowing O(1) lookup by E.164 phone number or email address.
- **subscribehandler**: Handles `customer_deleted` events by removing all contacts for that customer.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.contact-manager.request`. The `listenhandler` matches the URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/contacts$` | POST | Create contact |
| `/v1/contacts\?(.*)$` | GET | List contacts with pagination/filters |
| `/v1/contacts/{uuid}$` | GET, PUT, DELETE | Get / update / delete contact |
| `/v1/contacts/lookup\?(.*)$` | GET | Lookup by phone (E.164) or email |
| `/v1/contacts/{uuid}/phone-numbers$` | POST | Add phone number to contact |
| `/v1/contacts/{uuid}/phone-numbers/{uuid}$` | DELETE | Remove phone number |
| `/v1/contacts/{uuid}/emails$` | POST | Add email to contact |
| `/v1/contacts/{uuid}/emails/{uuid}$` | DELETE | Remove email |
| `/v1/contacts/{uuid}/tags$` | POST | Assign tag to contact |
| `/v1/contacts/{uuid}/tags/{uuid}$` | DELETE | Remove tag from contact |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.
