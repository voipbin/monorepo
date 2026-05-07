# Common Gotchas

Hard-won lessons from production incidents. Each entry below caused real failures; these are the patterns engineers most commonly trip over.

## Updating Shared Library Function Signatures

**CRITICAL: When updating function signatures in bin-common-handler, you MUST account for ALL call patterns across the monorepo.**

Services use different import aliases and call formats:
```go
// Some services use single-line with one alias
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, queueName, serviceName, "")

// Other services use multi-line with different alias
notifyHandler := commonnotify.NewNotifyHandler(
    sockHandler,
    reqHandler,
    queueName,
    serviceName,
)
```

**When updating function signatures:**
1. **Search for ALL import aliases** - Use `grep -r "notifyhandler\|commonnotify" --include="*.go"` to find all aliases
2. **Check for multi-line calls** - Simple sed patterns only match single-line; multi-line calls will be missed
3. **Run full verification for ALL services** - Don't rely on pattern matching; build errors will catch missed updates
4. **Prefer manual verification** - After bulk updates, manually review files with different patterns

**Example failure mode:**
```bash
# This sed command ONLY matches single-line patterns:
sed -i 's/notifyhandler.NewNotifyHandler(\([^)]*\))/notifyhandler.NewNotifyHandler(\1, "")/g'

# It MISSES multi-line patterns like:
commonnotify.NewNotifyHandler(
    sockHandler,
    reqHandler,
    queueName,
    serviceName,  # Missing 5th param!
)
```

**Safe approach:** After any bin-common-handler signature change, run the full verification workflow on ALL 34 services to catch any missed updates.

## Prometheus Metric Name Conflicts

**CRITICAL: Service-level `metricshandler` metrics MUST NOT reuse metric names already registered by `bin-common-handler/pkg/requesthandler`.**

The shared `requesthandler` registers these metrics (namespaced per service) via `initPrometheus()` when `NewRequestHandler()` is called:
- `<namespace>_request_process_time` — histogram of RPC request processing time
- `<namespace>_event_publish_total` — counter of published events by type

If a service's `metricshandler` package registers a metric with the same fully-qualified name (same namespace + name) but different labels or help text, `prometheus.MustRegister` will **panic at startup**, causing a **CrashLoopBackOff**.

```go
// ❌ WRONG — conflicts with requesthandler's event_publish_total
EventPublishTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "agent_manager",
        Name:      "event_publish_total",  // Already registered by requesthandler!
        Help:      "Total number of published events",
    },
    []string{"type"},  // Different labels → panic
)

// ✅ CORRECT — use a unique name
ServiceEventTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "agent_manager",
        Name:      "service_event_total",  // Unique name
        Help:      "Total number of service-level events",
    },
    []string{"type"},
)
```

**Before adding metrics to a service's `metricshandler`:** Check `bin-common-handler/pkg/requesthandler/main.go` `initPrometheus()` for existing metric names to avoid collisions.

## UUID Fields and DB Tags

**CRITICAL: UUID fields MUST use the `,uuid` db tag for proper type conversion.**

```go
// ✅ CORRECT - UUID field with uuid tag
type Model struct {
    ID         uuid.UUID `db:"id,uuid"`
    CustomerID uuid.UUID `db:"customer_id,uuid"`
}

// ❌ WRONG - Missing uuid tag
type Model struct {
    ID         uuid.UUID `db:"id"`  // Will cause conversion issues
}
```

**Why this matters:**

1. **Database queries fail silently** - Filters with UUID fields without `,uuid` tags are passed as strings instead of binary values, causing no database matches
2. **Type conversion errors** - `commondatabasehandler.PrepareFields()` needs the `,uuid` tag to convert `uuid.UUID` → binary for MySQL
3. **API bugs** - List endpoints return empty results even when data exists

**Example bug:**
```go
// Bug: conversation model missing uuid tags
type Conversation struct {
    CustomerID uuid.UUID `db:"customer_id"`  // Missing ,uuid tag
}

// Result: GET /v1/conversations?customer_id=<uuid> returns []
// Because filter is passed as string, not binary
```

**How to fix:**
1. Add `,uuid` tag to ALL uuid.UUID fields in model structs
2. Regenerate mocks: `go generate ./...`
3. Update tests: If tests mock database queries, verify UUID values are `uuid.UUID` type, not strings
4. Run verification workflow: `go mod tidy && go mod vendor && go generate ./... && go clean -testcache && go test ./...`

**Always verify UUID fields have `,uuid` tags when:**
- Adding new models
- Refactoring to use `commondatabasehandler` pattern
- Debugging empty API list responses
- Reviewing pull requests with model changes

## Model/Struct Changes Require OpenAPI Updates

**CRITICAL: Most API-facing structs are tightly coupled with OpenAPI specs. When changing struct definitions, you MUST also update the corresponding OpenAPI schema.**

This applies to any struct that:
- Is returned by API endpoints (response models)
- Is accepted by API endpoints (request models)
- Is embedded in other API-facing structs

**When modifying struct fields:**
1. **Update the Go struct** in the service's `models/` directory
2. **Update the OpenAPI schema** in `bin-openapi-manager/openapi/openapi.yaml`
3. **Regenerate OpenAPI types** with `go generate ./...` in `bin-openapi-manager`
4. **Regenerate API server code** with `go generate ./...` in `bin-api-manager` (it generates server code FROM the openapi.yaml)
5. **Run verification** for the service, `bin-openapi-manager`, AND `bin-api-manager`

**⚠️ IMPORTANT: Before modifying any OpenAPI schema, you MUST read and follow the AI-Native Specification Rules located in `bin-openapi-manager/CLAUDE.md`.**

**Why api-manager regeneration is required:**
`bin-api-manager` uses `go generate` to create server code directly from `bin-openapi-manager/openapi/openapi.yaml`. The generated code lives in `bin-api-manager/gens/openapi_server/gen.go`. If you update the OpenAPI spec but don't regenerate api-manager, the API server will use stale types.

**Example:**
```go
// Changing this in bin-talk-manager/models/message/message.go:
type Media struct {
    AgentID uuid.UUID `json:"agent_id,omitempty"`  // Changed from Agent amagent.Agent
}

// Requires updating bin-openapi-manager/openapi/openapi.yaml:
TalkManagerMedia:
  properties:
    agent_id:           # Changed from agent: $ref AgentManagerAgent
      type: string
      format: uuid

// Then regenerate BOTH:
// cd bin-openapi-manager && go generate ./...
// cd bin-api-manager && go generate ./...
```

**Why this matters:**
- Clients (web apps, mobile apps) depend on the OpenAPI spec for type generation
- The API server uses generated types from the spec
- Mismatched specs cause runtime errors or silent data loss
- API documentation becomes incorrect

**For complete gotcha explanations and troubleshooting, see [code-quality-standards.md#common-gotchas](../reference/code-quality-standards.md#common-gotchas)**

## Listener Wire-Format Mismatch (Silent No-Op)

**Symptom:** A POST or PUT to a public endpoint returns HTTP 200, but a follow-up GET shows nothing changed. No error logs anywhere.

**Root cause:** The `bin-common-handler` requesthandler client and the target service's listenhandler disagree on the JSON shape of the RPC body. `json.Unmarshal` silently ignores unknown top-level keys, so the listener parses a zero-valued struct, the handler operates on nil/zero fields, and the dynamic-update SQL writes nothing (or the create writes empty fields).

**Real incident (2026-05-08):** `PUT /v1.0/outbound_config` returned 200 for weeks but never persisted any change.

- Client (`bin-common-handler/pkg/requesthandler/call_outbound_configs.go`) marshaled the bare `outboundconfig.UpdateRequest`: `{"name":"...","codecs":"..."}`
- Listener (`bin-call-manager/pkg/listenhandler/v1_outbound_configs.go`) expected the body wrapped in `{"request":{...}}` because `V1DataOutboundConfigsIDPut` had `Request outboundconfig.UpdateRequest` as its only field.
- Result: `req.Request` was the zero `UpdateRequest{}` — every `*string` / `*[]string` pointer was nil. The DB layer treated nil as "no change," updated only `tm_update`, and returned the unchanged row with HTTP 200.

**Why nobody caught it sooner:**
- The listener test (`Test_processV1OutboundConfigsIDPut`) used `Update(gomock.Any(), tt.expectID, gomock.Any())` — `gomock.Any()` for the third argument meant the parsed payload was never validated. The test passed even when every field was zero.
- No client-side test existed for `CallV1OutboundConfigUpdate`, so the marshaled wire shape was never asserted.

### Prevention rules (already encoded in conventions)

1. **Listener request models must be flat.** No `Request` wrapper field in `pkg/listenhandler/models/request/*.go`. See [`../conventions/rpc.md`](../conventions/rpc.md) §9.5.
2. **Client must marshal via the listener model**, not via the domain model and not via an inline anonymous struct. See [`../conventions/rpc.md`](../conventions/rpc.md) §9.6.
3. **Listener tests must use a struct-literal matcher for parsed bodies** — never `gomock.Any()` for an argument the test parsed from `m.Data`. See [`../conventions/testing.md`](../conventions/testing.md) §13.8.
4. **Every requesthandler typed method needs a wire-shape test** — assert the exact `sock.Request.Data` bytes, plus a `strings.Contains` guard against accidental wrapper re-introduction. See [`../conventions/testing.md`](../conventions/testing.md) §13.9.

If you find yourself violating any of these "just for this one resource," go back and read the incident — every one of these rules exists because skipping it caused a silent production failure.

### Diagnostic procedure when symptom appears

1. **Confirm the HTTP layer is fine.** `kubectl logs -n bin-manager api-manager-* | grep -i "PUT.*<endpoint>"` should show 200. If non-200, the bug is upstream of the RPC.
2. **Check the listener request model.** Open `bin-<target>-manager/pkg/listenhandler/models/request/<resource>.go`. If a struct has a single `Request <DomainModel>` field, that's the smoking gun.
3. **Check the client marshaling.** Open `bin-common-handler/pkg/requesthandler/<service>_<resource>.go`. If it does `json.Marshal(req)` directly (instead of marshaling a `cmrequest.V1DataXxx` wrapper-or-flat struct), it's almost certainly mismatched.
4. **Inspect the listener test.** If the mock matcher is `Update(gomock.Any(), id, gomock.Any())` (or `Create(...)`), the test is too loose to have caught this.
5. **Reproduce locally** by writing a minimal test that unmarshals what the client actually sends into the listener's struct and prints the result — every nil pointer/zero field confirms the diagnosis.

## Feature Changes Require RST Documentation Updates

**CRITICAL: The RST docs in `bin-api-manager/docsdev/source/` are the primary user-facing documentation and the single source of truth for how the platform works. When adding or changing any user-visible feature, you MUST update the relevant RST docs.**

The RST documentation at `bin-api-manager/docsdev/source/` is what customers, developers, and integrators rely on to understand VoIPbin's APIs, billing, features, and behavior. If the docs don't reflect reality, users have no way to know a feature exists or how it works. Stale docs are worse than no docs — they actively mislead.

**This applies when you:**
- Add a new billable service type (update rate tables, diagrams, examples in `billing_account_overview.rst`)
- Add or modify API endpoints (update the relevant resource's `*_overview.rst`, `*_tutorial.rst`, `*_struct.rst`)
- Change pricing, rates, or billing behavior
- Add new event types that affect user-visible webhooks
- Add new resource types, statuses, or fields
- Change any behavior documented in the existing RST files

**When updating RST docs:**
1. **Edit the RST source** in `bin-api-manager/docsdev/source/`
2. **Clean rebuild the HTML**: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
3. **Force-add the build output**: `git add -f bin-api-manager/docsdev/build/` (root `.gitignore` excludes `build/`)
4. **Commit both RST source and built HTML together**

**IMPORTANT:** Always do a clean rebuild (`rm -rf build` first). Incremental Sphinx builds may miss cross-page references. The built HTML is tracked in git and must stay in sync with the RST sources.

**RST struct docs must match `WebhookMessage`, not internal model structs.**
The `WebhookMessage` struct (defined in `models/<entity>/webhook.go`) determines exactly which fields are exposed to external users via the API. RST struct documentation (`*_struct_*.rst`) must only include fields present in `WebhookMessage`. Do not document internal-only fields (e.g., `PodID`, `Username`, `PermissionIDs`) that are stripped by `ConvertWebhookMessage()`. When verifying RST accuracy, always compare against `WebhookMessage` fields, not the internal model struct.

**After making user-facing changes**, also verify RST docs in `bin-api-manager/docsdev/source/` are in sync with the code. Compare struct docs against the relevant `WebhookMessage` fields (in `models/<entity>/webhook.go`), not the internal model struct. If RST updates are needed, rebuild HTML: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` and force-add: `git add -f bin-api-manager/docsdev/build/`.

**Why this matters:**
- RST docs are the single source of truth for external users
- Customers cannot discover undocumented features
- Stale rate tables cause billing confusion and support tickets
- The built HTML is deployed directly — if it's not committed, the live site is out of date
