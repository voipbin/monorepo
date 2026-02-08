# Service Agents Contacts Endpoints Design

## Purpose

Add `/service_agents/contacts/` endpoints to expose contact management to VoIPbin Talk agents. Currently, contact CRUD is only available through the `/contacts/` endpoints which require `PermissionCustomerAdmin` or `PermissionCustomerManager`. Talk agents need direct access to contacts without elevated permissions.

## Endpoints

All endpoints are scoped to the authenticated agent's `customer_id`. Any authenticated agent can access contacts belonging to their organization.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/service_agents/contacts` | List contacts (paginated, filtered) |
| POST | `/service_agents/contacts` | Create a contact |
| GET | `/service_agents/contacts/{contact_id}` | Get a contact |
| PUT | `/service_agents/contacts/{contact_id}` | Update a contact |
| DELETE | `/service_agents/contacts/{contact_id}` | Delete a contact |
| GET | `/service_agents/contacts/lookup` | Lookup by phone/email |
| POST | `/service_agents/contacts/{contact_id}/phone_numbers` | Add phone number |
| PUT | `/service_agents/contacts/{contact_id}/phone_numbers/{phone_number_id}` | Update phone number |
| DELETE | `/service_agents/contacts/{contact_id}/phone_numbers/{phone_number_id}` | Delete phone number |
| POST | `/service_agents/contacts/{contact_id}/emails` | Add email |
| PUT | `/service_agents/contacts/{contact_id}/emails/{email_id}` | Update email |
| DELETE | `/service_agents/contacts/{contact_id}/emails/{email_id}` | Delete email |
| POST | `/service_agents/contacts/{contact_id}/tags` | Add tag |
| DELETE | `/service_agents/contacts/{contact_id}/tags/{tag_id}` | Remove tag |

## Key Differences from `/contacts/`

- **Permissions:** No `PermissionCustomerAdmin`/`PermissionCustomerManager` required. Any authenticated agent can access.
- **Scoping:** Contacts filtered by agent's `customer_id` (all customer contacts visible to any agent).
- **Request/response schemas:** Identical to existing `/contacts/` endpoints. Reuses `ContactManagerContact`, `ContactManagerPhoneNumber`, `ContactManagerEmail`, etc.

## Implementation Architecture

### Layer 1 - OpenAPI Spec (`bin-openapi-manager`)

New path files under `bin-openapi-manager/openapi/paths/service_agents/`:

| File | Endpoints |
|------|-----------|
| `contacts.yaml` | GET (list) + POST (create) |
| `contacts_{contact_id}.yaml` | GET + PUT + DELETE |
| `contacts_lookup.yaml` | GET (lookup by phone/email) |
| `contacts_{contact_id}_phone_numbers.yaml` | POST (add) |
| `contacts_{contact_id}_phone_numbers_{phone_number_id}.yaml` | PUT + DELETE |
| `contacts_{contact_id}_emails.yaml` | POST (add) |
| `contacts_{contact_id}_emails_{email_id}.yaml` | PUT + DELETE |
| `contacts_{contact_id}_tags.yaml` | POST (add) |
| `contacts_{contact_id}_tags_{tag_id}.yaml` | DELETE |

These files reuse existing request/response schemas from the `/contacts/` spec. No new models are needed.

### Layer 2 - HTTP Handlers (`bin-api-manager/server/`)

New file: `service_agents_contacts.go`

Follows the same pattern as other `service_agents_*.go` handlers:
1. Extract agent from JWT context
2. Call service handler method
3. Return response

### Layer 3 - Service Handler (`bin-api-manager/pkg/servicehandler/`)

New file: `service_agent_contact.go`

Methods like `ServiceAgentContactGet`, `ServiceAgentContactCreate`, etc. These:
- Validate agent authentication (agent extracted from JWT)
- Scope by `customer_id` from the authenticated agent
- Delegate to existing `reqHandler.ContactV1*` RPC calls
- No admin/manager permission checks

## Implementation Sequence

1. **OpenAPI path files** - Create 9 path YAML files, register in `openapi.yaml`
2. **Regenerate OpenAPI types** - `go generate ./...` in `bin-openapi-manager`
3. **Regenerate API server** - `go generate ./...` in `bin-api-manager` to update `gen.go`
4. **Service handler methods** - Create `service_agent_contact.go` in `pkg/servicehandler/`
5. **HTTP handlers** - Create `service_agents_contacts.go` in `server/`
6. **Verification** - Full workflow for `bin-openapi-manager` and `bin-api-manager`

## Services Affected

- `bin-openapi-manager` - New OpenAPI path files
- `bin-api-manager` - New handlers and service handler methods

No changes to `bin-contact-manager` or `bin-common-handler` since we reuse existing RPC calls.
