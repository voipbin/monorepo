# bin-openapi-manager Usage

## Consuming Generated Types

Every `bin-*-manager` service imports the generated models package:

```go
import "monorepo/bin-openapi-manager/gens/models"

// Example usage
var agent models.AgentManagerAgent
var status models.AgentManagerAgentStatus = models.AgentManagerAgentStatusAvailable
```

The `go.mod` in each consumer declares a `replace` directive:

```
replace monorepo/bin-openapi-manager => ../bin-openapi-manager
```

**Type naming convention:** `<ServiceName><ResourceType>` — e.g., `CallManagerCall`, `ConferenceManagerConference`. Enums follow `<TypeName><ValueName>` — e.g., `CallManagerCallStatusActive`.

### Pagination pattern

All list responses use cursor-based pagination with `CommonPagination`:

```go
type CommonPagination struct {
    NextPageToken string `json:"next_page_token"`
    Result        []...  // typed per-resource
}
```

Query parameters: `page_size` (count) and `page_token` (cursor, typically a `tm_create` timestamp).

### Schema validation against WebhookMessage

OpenAPI schemas must match the `WebhookMessage` struct in each service's `models/<resource>/webhook.go` — not the internal model struct. Only fields present in `WebhookMessage` (after `ConvertWebhookMessage()`) should appear in the OpenAPI schema. Internal-only fields (`PodID`, `Username`, etc.) must not be documented.

Mapping example:
```
bin-call-manager/models/call/webhook.go → WebhookMessage
    → openapi.yaml → CallManagerCall schema
```

## Regeneration

### After editing the OpenAPI spec

```bash
# In bin-openapi-manager directory:

# Validate the spec (fails fast on YAML errors)
oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null

# Regenerate Go types
go generate ./...

# The generated file is gens/models/gen.go — commit it with your spec changes
```

### After adding a new resource

1. Create `openapi/paths/<new-resource>/main.yaml` (collection endpoints).
2. Create `openapi/paths/<new-resource>/id.yaml` (individual resource endpoints).
3. Add path references in `openapi/openapi.yaml` under `paths:`.
4. Add schema definitions in `openapi/openapi.yaml` under `components/schemas:`.
5. Run `go generate ./...`.
6. Run `go build ./...` in `bin-api-manager` to verify no type breakage.

### After changing an existing schema

1. Edit the schema in `openapi/openapi.yaml` or the relevant path file.
2. Run `go generate ./...` in `bin-openapi-manager`.
3. Check if the generated type changed shape (diff `gens/models/gen.go`).
4. If the shape changed, audit all consumer services for compile errors.
5. Run `go build ./...` in `bin-api-manager` (the primary consumer).
6. Update RST docs in `bin-api-manager/docsdev/source/` if the change affects public API behavior.

### Commit checklist

- [ ] `openapi/openapi.yaml` (and/or path files) updated
- [ ] `go generate ./...` run successfully
- [ ] `gens/models/gen.go` committed alongside spec changes
- [ ] Dependent services compile (at minimum `bin-api-manager`)
- [ ] RST docs updated if this is a user-visible API change
