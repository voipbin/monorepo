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
| `tm_delete` | timestamp | Soft-delete sentinel; active = `9999-01-01` |

Table: `contact_manager_contact`

### PhoneNumber

Child record linking a phone number in E.164 format to a contact. A contact may have up to N phone numbers.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `contact_id` | UUID | Parent contact |
| `phone_number` | string | E.164 format (e.g., `+15551234567`) |
| `tm_create` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel |

Table: `contact_manager_phone_number`

### Email

Child record linking an email address to a contact.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `contact_id` | UUID | Parent contact |
| `email` | string | |
| `tm_create` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel |

Table: `contact_manager_email`

### TagAssignment

Many-to-many link between contacts and tags managed by `bin-tag-manager`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `contact_id` | UUID | |
| `tag_id` | UUID | References tag in bin-tag-manager |
| `tm_create` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel |

Table: `contact_manager_tag_assignment`

## Key Business Rules

1. **Tenant isolation**: All queries filter by `customer_id`. A contact is never visible to another customer.

2. **Soft deletes**: Delete operations set `tm_delete` to the current timestamp. Queries always add `tm_delete = '9999-01-01 00:00:00.000000'` to active-record filters.

3. **Phone lookup**: The `GET /v1/contacts/lookup?customer_id=<uuid>&phone_e164=<e164>` endpoint uses a Redis cache index for O(1) lookup. Cache is populated on contact creation and invalidated on update/delete.

4. **Email lookup**: Similarly indexed in Redis. Used for inbound email matching.

5. **Cascading delete**: When a `customer_deleted` event is received from `bin-customer-manager`, all contacts (and their phone numbers, emails, and tag assignments) for that customer are deleted.

6. **Tag ownership**: Tags are defined in `bin-tag-manager`. This service only stores the assignment link. Deleting a contact also removes all tag assignments for that contact.

7. **Multiple addresses**: A single contact may have multiple phone numbers and emails. These are managed as independent child records with their own UUIDs.

8. **Event publishing**: Create, update, and delete operations publish `contact_created`, `contact_updated`, and `contact_deleted` events to `QueueNameContactEvent`. Downstream services (e.g., call flows) can subscribe to keep caller-ID caches current.
