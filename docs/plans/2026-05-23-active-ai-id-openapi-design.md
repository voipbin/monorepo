# Design: Add active_ai_id to AIManagerMessage OpenAPI schema

**Date:** 2026-05-23
**Branch:** NOJIRA-Add-active-ai-id-to-openapi
**Scope:** OpenAPI spec field addition only — no new endpoints, no handler changes, no DB migrations

---

## Background

PR #931 (`NOJIRA-Add-active-ai-id-to-aimessage`) added `active_ai_id` to the backend:
- `bin-ai-manager/models/message/webhook.go`: `ActiveAIID uuid.UUID json:"active_ai_id,omitempty"` (both `WebhookMessage` and `IntermediateWebhookMessage` structs)

However, the field was NOT added to `bin-openapi-manager/openapi/openapi.yaml` `AIManagerMessage` schema. The REST API response therefore includes the field, but it has no official contract. This PR adds the missing spec entry and re-runs codegen.

## Source of truth

```
bin-ai-manager/models/message/webhook.go line 17:
    ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty"`
```

Field is `omitempty` — absent when zero-value (`00000000-0000-0000-0000-000000000000`).

The RST doc (`bin-api-manager/docsdev/source/ai_struct_message.rst`) already documents `active_ai_id` at lines 18 and 31 — it was added ahead of the spec. No RST changes needed.

## Change

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Location:** `components.schemas.AIManagerMessage`, after `tool_call_id`, before `tm_create`

```yaml
        active_ai_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the AI configuration that was active when this message was created. Returned from the `GET /ais` response."
          example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
```

**Why `x-go-type: string`:** All other UUID fields in this schema (`id`, `customer_id`, `aicall_id`, `activeflow_id`) use `x-go-type: string` to emit `string` instead of `openapi_types.UUID` in gen.go. Consistent treatment required.

**Why `omitempty` is not in YAML:** OpenAPI `omitempty` semantics are controlled by the JSON tag in the backend struct, not the spec. The spec correctly marks the field as non-required (no entry in `required:` array).

## Codegen result

After `go generate ./...` in `bin-openapi-manager`:

```go
// gen.go (AIManagerMessage struct)
ActiveAiId *string `json:"active_ai_id,omitempty"`
```

Pointer type (`*string`) is correct — oapi-codegen emits pointer for optional object fields.

## Verification steps

```bash
cd bin-openapi-manager
go generate ./...
grep "ActiveAiId" gens/models/gen.go   # must appear

cd ../bin-api-manager
go build ./...                          # must succeed, zero errors
```

## Scope: what is NOT changed

- No new path/endpoint YAML files
- No handler code in `bin-api-manager` or `bin-ai-manager`
- No DB schema or Alembic migration
- No RST doc changes (already present)
- No changes to any other service

## Files changed

| File | Change |
|---|---|
| `bin-openapi-manager/openapi/openapi.yaml` | Add `active_ai_id` field to `AIManagerMessage` schema |
| `bin-openapi-manager/gens/models/gen.go` | Regenerated — `ActiveAiId *string` added to `AIManagerMessage` struct |
