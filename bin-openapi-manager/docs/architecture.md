# bin-openapi-manager Architecture

This is a **Class E** service: code-generation only. There is no runtime binary, no RabbitMQ listener, and no deployable container. It is consumed at compile time by every other `bin-*-manager` service.

## Codegen Pipeline

The pipeline converts a modular OpenAPI 3.0 YAML specification into Go type definitions using `oapi-codegen`.

```
openapi/openapi.yaml          ← main spec (aggregates all paths and schemas)
openapi/paths/<resource>/     ← modular path definitions ($ref'd from openapi.yaml)
        |
        v  (go generate ./...)
configs/config_model/
  generate.go                 ← go:generate directive
  config.generate.yaml        ← oapi-codegen config (package: models, generate: models)
        |
        v
gens/models/gen.go            ← generated Go types (DO NOT EDIT manually)
```

**Step-by-step:**

1. Edit `openapi/openapi.yaml` or a file under `openapi/paths/`.
2. Run `go generate ./...` in `bin-openapi-manager`.
3. `generate.go` invokes: `oapi-codegen -config configs/config_model/config.generate.yaml`.
4. `gens/models/gen.go` is rewritten with updated type definitions.
5. Rebuild every consuming service to verify no type breakage.

### Path organization

```
openapi/paths/
├── agents/          # GET/POST /agents, GET/PUT/DELETE /agents/{id}, ...
├── calls/           # Call endpoints
├── conferences/     # Conference endpoints
├── files/           # File storage endpoints
└── ...              # ~50 resource directories total
```

Within each directory:
- `main.yaml` — collection endpoints (list, create)
- `id.yaml` — individual resource endpoints (get, update, delete)
- `id_<action>.yaml` — specific sub-actions (e.g., `id_recording_start.yaml`)

### Components section

The `components/schemas` section in `openapi.yaml` defines all types, prefixed by service name to avoid conflicts:
- `AgentManager*` — agent types
- `CallManager*` — call types
- `AIManager*` — AI engine types
- `StorageManager*` — file storage types
- `ConferenceManager*` — conference types
- (and one prefix per manager service)

### oapi-codegen configuration

```yaml
# configs/config_model/config.generate.yaml
package: models
generate:
  models: true
output-options:
  skip-prune: true
```

Only model types are generated — no server stubs, no client code.

## Output Artifacts

| Artifact | Location | Notes |
|---------|---------|-------|
| Generated Go types | `gens/models/gen.go` | Auto-generated; do not edit manually |
| OpenAPI YAML spec | `openapi/openapi.yaml` | Hand-authored; aggregates all path files |
| Path definitions | `openapi/paths/<resource>/` | Modular YAML files; referenced via `$ref` |
| Generator config | `configs/config_model/config.generate.yaml` | Controls oapi-codegen output |

### Key generation rules (AI-native OpenAPI)

These rules apply to all new or modified fields in the spec:

1. **Use `oneOf` for polymorphism** — never `additionalProperties: true` for type-discriminated objects.
2. **Format structured strings** — UUIDs use `format: uuid`; timestamps use `format: date-time`; phone numbers use a `pattern`; enums use the `enum` keyword.
3. **Provenance in descriptions** — ID fields referencing another resource must state the source endpoint: `"The X returned from POST /y response."`
4. **Realistic examples on every leaf property** — never use `"string"`, `null`, or generic placeholders.
5. **`minItems` on required arrays** — arrays that must be non-empty should declare `minItems: 1`.

Violating these rules makes the spec less useful to AI agents consuming the API and may cause downstream build failures when generated types change shape unexpectedly.
