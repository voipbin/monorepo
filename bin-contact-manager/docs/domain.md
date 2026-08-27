# bin-contact-manager — Domain

## Domain Entities

### Contact

The primary entity. Belongs to a `customer_id` and represents a person or organisation in the CRM.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `first_name` | string | |
| `last_name` | string | |
| `display_name` | string | Derived or explicit |
| `company` | string | |
| `job_title` | string | |
| `source` | string | Origin system (e.g., `crm`, `import`) |
| `external_id` | string | ID in external system |
| `notes` | string | Free-text |
| `tm_create` | timestamp | Creation time |
| `tm_update` | timestamp | Last update time |
| `tm_delete` | timestamp | Soft-delete marker; active = `NULL`, deletion records the actual timestamp |

Table: `contact_contacts`

### PhoneNumber

Reverse-projection of a `contact_addresses` row with `type='tel'`. A contact may have up to N phone numbers. The address store is the source of truth (VOIP-1207); the `PhoneNumber` model is a read-time view (`Number` = the normalized `target`).

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | `contact_addresses.id` |
| `contact_id` | UUID | Parent contact |
| `number` | string | E.164 format (e.g., `+155****4567`); from `target` |
| `is_primary` | bool | One primary per contact across ALL address types |
| `tm_create` | timestamp | |

Table: `contact_addresses` (`type='tel'`). Hard-delete (no `tm_delete`).

### Email

Reverse-projection of a `contact_addresses` row with `type='email'`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | `contact_addresses.id` |
| `contact_id` | UUID | Parent contact |
| `address` | string | Lowercased; from `target` |
| `is_primary` | bool | One primary per contact across ALL address types |
| `tm_create` | timestamp | |

Table: `contact_addresses` (`type='email'`). Hard-delete (no `tm_delete`).

### TagAssignment

Many-to-many link between contacts and tags managed by `bin-tag-manager`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `contact_id` | UUID | |
| `tag_id` | UUID | References tag in bin-tag-manager |
| `tm_create` | timestamp | |
| `tm_delete` | timestamp | Soft-delete marker (active = `NULL`) |

Table: `contact_tag_assignments`

### Case event payloads (`models/kase`, `models/casenote`)

Internal, agent-facing state changes on a Case are published through the plain `PublishEvent()` primitive (never `PublishWebhookEvent()` — a CaseNote must never reach a customer-facing webhook). VOIP-1405 replaced the ad-hoc `map[string]uuid.UUID` payloads of these sites with named structs; the JSON key SET is unchanged (only field order differs, which is JSON-semantically irrelevant), so fanout consumers see no change.

| Event type | Constant | Payload type | JSON keys | Subscription address |
|---|---|---|---|---|
| `case_note_created` | `casenote.EventTypeCaseNoteCreated` | `*casenote.CaseNote` | full note | `case_id` (override) |
| `case_note_deleted` | `casenote.EventTypeCaseNoteDeleted` | `*casenote.CaseNoteDeletedEvent` | `id`, `case_id`, `customer_id` | `case_id` (override) |
| `case_tag_added` | `kase.EventTypeCaseTagAdded` | `*kase.CaseTagEvent` | `case_id`, `tag_id` | `case_id` (override) |
| `case_tag_removed` | `kase.EventTypeCaseTagRemoved` | `*kase.CaseTagEvent` | `case_id`, `tag_id` | `case_id` (override) |
| `case_contact_attributed` | `kase.EventTypeCaseContactAttributed` | `*kase.CaseContactEvent` | `case_id`, `contact_id` | `case_id` (override) |
| `case_contact_detached` | `kase.EventTypeCaseContactDetached` | `*kase.CaseContactEvent` | `case_id`, `contact_id` (`uuid.Nil`) | `case_id` (override) |

`EventSubscriptionID()` is implemented with a POINTER receiver on all four payload types (`*CaseNote`, `*CaseNoteDeletedEvent`, `*CaseTagEvent`, `*CaseContactEvent`): the event data reaches `notifyhandler` as a pointer and the interface assertion matches the dynamic type, so a value receiver would silently never be picked up. `kase.Case` itself deliberately carries NO override — its own id already is the address.

`CaseNote` and `CaseNoteDeletedEvent` both carry their own top-level `id`; the override is what keeps the address on the case axis instead of that note id. Without it the events would publish under a well-formed but unbindable address that no runtime metric can flag, which is why `models/kase/routingkey_golden_test.go` pins the `case_note_deleted` address explicitly.

## Key Business Rules

1. **Tenant isolation**: All queries filter by `customer_id`. A contact is never visible to another customer.

2. **Soft deletes**: Delete operations set `tm_delete` to the current timestamp. Active records have `tm_delete IS NULL`; queries add `tm_delete IS NULL` to active-record filters.

3. **Phone lookup**: The `GET /v1/contacts/lookup?customer_id=<uuid>&phone_e164=<e164>` endpoint uses a Redis cache index for O(1) lookup. Cache is populated on contact creation and invalidated on update/delete.

4. **Email lookup**: Similarly indexed in Redis. Used for inbound email matching.

5. **Cascading delete**: When a `customer_deleted` event is received from `bin-customer-manager`, all contacts (and their phone numbers, emails, and tag assignments) for that customer are deleted.

6. **Tag ownership**: Tags are defined in `bin-tag-manager`. This service only stores the assignment link. Deleting a contact also removes all tag assignments for that contact.

7. **Multiple addresses**: A single contact may have multiple phone numbers and emails. These are managed as independent child records with their own UUIDs.

8. **Event publishing**: Create, update, and delete operations publish `contact_created`, `contact_updated`, and `contact_deleted` events to `QueueNameContactEvent`. Downstream services (e.g., call flows) can subscribe to keep caller-ID caches current. The payload is the `*contact.WebhookMessage` value produced by `ConvertWebhookMessage()`. **VOIP-1405 payload change**: this used to be the `[]byte` returned by `CreateWebhookEvent()`, which `PublishEvent` then marshaled a second time, storing the event as a base64 JSON *string*. The published payload is now a JSON object. This is an intended, independent improvement and is NOT reverted by disabling the global topic exchange option.

9. **Case events**: see the "Case event payloads" table above for the six internal case events, their named constants, and their subscription addresses on the global topic exchange.
