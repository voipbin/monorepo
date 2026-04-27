# WebhookMessage Pattern for External API Responses (MANDATORY)

**CRITICAL: All external-facing API responses MUST use the `WebhookMessage` pattern. Never return raw internal model structs directly to external clients.**

Internal model structs (e.g., `Speaking`, `Call`, `Recording`) may contain infrastructure details, internal routing fields, or implementation-specific data that must not be exposed. The `WebhookMessage` struct serves as the external-facing representation.

**Pattern:**
1. Define `WebhookMessage` in `models/<entity>/webhook.go` — includes only fields safe for external clients
2. Add `ConvertWebhookMessage()` method on the internal struct
3. In `bin-api-manager/pkg/servicehandler/`, call `.ConvertWebhookMessage()` before returning to the HTTP layer
4. The private helper (e.g., `speakingGet()`) returns the internal struct for internal use (routing, permission checks)
5. The public method (e.g., `SpeakingGet()`) returns `*WebhookMessage` for the API response

**Example:**
```go
// Private — returns internal struct with all fields (e.g., PodID for routing)
func (h *serviceHandler) speakingGet(ctx context.Context, id uuid.UUID) (*tmspeaking.Speaking, error) { ... }

// Public — returns WebhookMessage without internal fields
func (h *serviceHandler) SpeakingGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.WebhookMessage, error) {
    tmp, err := h.speakingGet(ctx, id)
    ...
    return tmp.ConvertWebhookMessage(), nil
}
```

**Compound result structs (e.g., `SignupResult`, `EmailVerifyResult`):**
- If a result struct embeds an internal model (e.g., `*Customer`), it MUST also have a `WebhookMessage` variant
- The variant replaces the internal model with its `*WebhookMessage` counterpart
- Example: `SignupResult{Customer *Customer}` → `SignupResultWebhookMessage{Customer *WebhookMessage}`
- The `ConvertWebhookMessage()` method must recursively convert embedded models

**When adding a new API resource:**
- Create `webhook.go` alongside the model definition
- Omit any fields that are infrastructure-specific or internal-only
- Update the OpenAPI schema to match `WebhookMessage` fields (not the internal struct)

## Examples in the codebase

- `bin-api-manager/pkg/servicehandler/flow.go` — `FlowGet` (line 145) and `FlowDelete` (line 84) wrap the private `flowGet` (line 20) and return `*fmflow.WebhookMessage`.
- `bin-api-manager/pkg/servicehandler/campaigns.go` — `CampaignGet` (line 137) and `CampaignGetsByCustomerID` (line 93) convert via `f.ConvertWebhookMessage()` before returning.
- `bin-api-manager/pkg/servicehandler/queue.go` — `QueueGet` (line 37) and `QueueList` (line 68) follow the same private/public split with `.ConvertWebhookMessage()` at the boundary.

## Why the pattern exists

Internal model structs intentionally carry fields that must not leak to external clients:

- **Routing identity** — e.g., `PodID`, `HostID` used to route follow-up RPCs to a specific pod (see [per-pod-queues.md](per-pod-queues.md)).
- **Auth internals** — e.g., `Username`, `PermissionIDs` on agent records.
- **Implementation hints** — e.g., session keys, internal flags used for cache coherency.

Returning the internal struct directly would expose all of these. `WebhookMessage` is the bottleneck where the API contract is enforced — and it pairs with the RST documentation rule: `*_struct_*.rst` files in `bin-api-manager/docsdev/source/` MUST document `WebhookMessage` fields, not the internal struct, so the public docs match what the API actually returns.
