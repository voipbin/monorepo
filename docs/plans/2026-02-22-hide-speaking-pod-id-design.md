# Hide Speaking pod_id from External API Responses

## Problem

The `Speaking` struct in `bin-tts-manager` includes a `PodID` field that exposes Kubernetes pod identity to external API clients. This field is serialized with `json:"pod_id"` and returned in all 7 speaking API endpoints. External clients should not see internal infrastructure details.

The `PodID` is used internally for pod-targeted RabbitMQ routing — `Say`, `Flush`, and `Stop` requests must be routed to the specific pod running the streaming session. This internal use must be preserved.

## Approach

Adopt the `WebhookMessage` pattern already established across the codebase (call, flow, campaign, recording, etc.). This pattern separates internal model structs from external-facing response structs.

### Why not `json:"-"`?

The `Speaking` struct is JSON-serialized for internal RabbitMQ RPC communication between `bin-tts-manager` and `bin-api-manager`. The `requesthandler` deserializes the JSON response into a `Speaking` struct and reads `PodID` for routing. Using `json:"-"` would break this internal communication path.

## Changes

### 1. New: `bin-tts-manager/models/speaking/webhook.go`

Create a `WebhookMessage` struct that mirrors `Speaking` but omits `PodID`:

```go
type WebhookMessage struct {
    commonidentity.Identity
    ReferenceType streaming.ReferenceType `json:"reference_type,omitempty"`
    ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
    Language      string                  `json:"language,omitempty"`
    Provider      string                  `json:"provider,omitempty"`
    VoiceID       string                  `json:"voice_id,omitempty"`
    Direction     streaming.Direction     `json:"direction,omitempty"`
    Status        Status                  `json:"status,omitempty"`
    TMCreate      *time.Time              `json:"tm_create,omitempty"`
    TMUpdate      *time.Time              `json:"tm_update,omitempty"`
    TMDelete      *time.Time              `json:"tm_delete,omitempty"`
}
```

Add `ConvertWebhookMessage()` method on `Speaking` and `CreateEventData()` for event marshaling (following the call model pattern).

### 2. Modify: `bin-api-manager/pkg/servicehandler/speaking.go`

Change public methods to return `*tmspeaking.WebhookMessage` instead of `*tmspeaking.Speaking`. Call `ConvertWebhookMessage()` before returning. The private `speakingGet()` remains unchanged — it returns `*Speaking` with `PodID` intact for internal routing.

### 3. Modify: `bin-openapi-manager/openapi/openapi.yaml`

Remove the `pod_id` property from the `TtsManagerSpeaking` schema.

### 4. Add rule to root `CLAUDE.md`

Add a rule under Code Quality requiring all external-facing API responses to use the `WebhookMessage` pattern. Internal model structs must never be returned directly to external clients.

### 5. Regenerate

- `bin-openapi-manager`: `go generate ./...`
- `bin-api-manager`: `go generate ./...`

## Data Flow (After Change)

```
tts-manager (Speaking with PodID)
    → JSON (includes pod_id) via RabbitMQ
    → api-manager requesthandler (deserializes Speaking with PodID)
    → api-manager servicehandler (uses PodID for routing, then calls ConvertWebhookMessage())
    → api-manager server (returns WebhookMessage without pod_id to HTTP client)
```

## Files Affected

| File | Action |
|------|--------|
| `bin-tts-manager/models/speaking/webhook.go` | Create |
| `bin-api-manager/pkg/servicehandler/speaking.go` | Modify return types |
| `bin-openapi-manager/openapi/openapi.yaml` | Remove pod_id property |
| `CLAUDE.md` (root) | Add WebhookMessage rule |
| `bin-openapi-manager/gens/models/gen.go` | Regenerate |
| `bin-api-manager/gens/openapi_server/gen.go` | Regenerate |
