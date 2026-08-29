# bin-contact-manager — Architecture

## Component Overview

`bin-contact-manager` is a Class A Go RPC microservice that manages CRM-style contact records for the VoIPbin platform. It stores contacts belonging to customers, with support for multiple phone numbers, emails, and tag assignments. It also serves O(1) lookup requests used by inbound call flows for caller-ID enrichment.

**Binary:** `contact-manager` (daemon) + `contact-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/contact-manager` | Daemon entry point; wires config, DB, cache, and handlers |
| `cmd/contact-control` | CLI tool for direct DB/cache management (bypasses RabbitMQ) |
| `cmd/case-control` | CLI tool for case management (no NotifyHandler; publishes nothing) |
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

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.contact-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 1 pattern total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `customer-manager.customer.*.deleted` | Customer deletion — cascade-deletes all contacts of the customer |

As of VOIP-1407 this topic-pattern binding is the **sole intake mechanism**; the old per-service fanout subscription (`QueueSubscribe` to `bin-manager.customer-manager.event`) has been removed from `Run()` entirely, along with the fanout-unbind step that used to follow a successful topic bind. A topic bind failure is now fatal to `Run()` — there is no fanout fallback left to degrade to.

## Event Publishing — Global Topic Exchange (VOIP-1404 / VOIP-1405 / VOIP-1407)

Both `cmd/contact-manager` and `cmd/contact-control` construct their `NotifyHandler` with `notifyhandler.WithGlobalTopicPublish()`. **As of VOIP-1407, this is the sole publish path**: the previous fanout publish to `QueueNameContactEvent` is no longer made, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) that exchange will eventually be deleted from the broker. Every event publishes to the global topic exchange `bin-manager.event` with the routing key:

```
<publisher>.<resource>.<subscription-id>.<action>
contact-manager.contact.<contact-id>.created
contact-manager.case.<case-id>.note_deleted
```

Rules that apply to this service:

- **Both binaries must carry the option.** They publish to the same logical stream, so enabling it on only one would leave consumers with gaps. `cmd/case-control` constructs no `NotifyHandler` and publishes nothing, so it is out of scope.
- **This service's publish-side behavior change comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update** (its own consumer-side subscribehandler code also changed separately for VOIP-1407, see the Event Subscriptions section above); a topic publish failure now propagates to the caller as an error (previously it was swallowed silently).
- **Subscription addresses**: contact lifecycle events are addressed by the contact's own id (default resolution from the payload's top-level `id`). Every case-scoped event (`case_note_*`, `case_tag_*`, `case_contact_*`) is addressed by the **case id** via `EventSubscriptionID()` overrides, so one `contact-manager.case.<case-id>.#` binding follows a whole case. See [domain.md](domain.md) for the payload types.
- **Rollback** is per binary: remove the single option argument. It does NOT revert the payload normalization described in domain.md.

The golden routing-key table for every event type this service publishes lives in `models/kase/routingkey_golden_test.go`.

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
