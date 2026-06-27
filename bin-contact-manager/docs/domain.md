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

## Key Business Rules

1. **Tenant isolation**: All queries filter by `customer_id`. A contact is never visible to another customer.

2. **Soft deletes**: Delete operations set `tm_delete` to the current timestamp. Active records have `tm_delete IS NULL`; queries add `tm_delete IS NULL` to active-record filters.

3. **Phone lookup**: The `GET /v1/contacts/lookup?customer_id=<uuid>&phone_e164=<e164>` endpoint uses a Redis cache index for O(1) lookup. Cache is populated on contact creation and invalidated on update/delete.

4. **Email lookup**: Similarly indexed in Redis. Used for inbound email matching.

5. **Cascading delete**: When a `customer_deleted` event is received from `bin-customer-manager`, all contacts (and their phone numbers, emails, and tag assignments) for that customer are deleted.

6. **Tag ownership**: Tags are defined in `bin-tag-manager`. This service only stores the assignment link. Deleting a contact also removes all tag assignments for that contact.

7. **Multiple addresses**: A single contact may have multiple phone numbers and emails. These are managed as independent child records with their own UUIDs.

8. **Event publishing**: Create, update, and delete operations publish `contact_created`, `contact_updated`, and `contact_deleted` events to `QueueNameContactEvent`. Downstream services (e.g., call flows) can subscribe to keep caller-ID caches current.
