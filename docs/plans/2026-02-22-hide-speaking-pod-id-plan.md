# Hide Speaking pod_id Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the `pod_id` field from external Speaking API responses while preserving internal pod-targeted routing.

**Architecture:** Create a `WebhookMessage` struct (established codebase convention) in `bin-tts-manager/models/speaking/` that omits `PodID`. Update `bin-api-manager` servicehandler to convert `Speaking` → `WebhookMessage` before returning to the HTTP layer. Remove `pod_id` from the OpenAPI spec.

**Tech Stack:** Go, OpenAPI 3.0, oapi-codegen, RabbitMQ (internal), Gin (HTTP)

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/`

---

### Task 1: Create WebhookMessage and ConvertWebhookMessage

**Files:**
- Create: `bin-tts-manager/models/speaking/webhook.go`

**Step 1: Create the webhook.go file**

Create `bin-tts-manager/models/speaking/webhook.go` with the following content:

```go
package speaking

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Speaking session.
// It omits internal fields like PodID that should not be exposed to API clients.
type WebhookMessage struct {
	commonidentity.Identity

	ReferenceType streaming.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
	Language      string                  `json:"language,omitempty"`
	Provider      string                  `json:"provider,omitempty"`
	VoiceID       string                  `json:"voice_id,omitempty"`
	Direction     streaming.Direction     `json:"direction,omitempty"`
	Status        Status                  `json:"status,omitempty"`

	TMCreate *time.Time `json:"tm_create,omitempty"`
	TMUpdate *time.Time `json:"tm_update,omitempty"`
	TMDelete *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts a Speaking to its external-facing representation.
func (h *Speaking) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,
		Language:      h.Language,
		Provider:      h.Provider,
		VoiceID:       h.VoiceID,
		Direction:     h.Direction,
		Status:        h.Status,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates webhook event data.
func (h *Speaking) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
```

**Step 2: Verify bin-tts-manager builds**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-tts-manager
go build ./cmd/...
```
Expected: clean build, no errors.

**Step 3: Run bin-tts-manager tests**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-tts-manager
go test ./...
```
Expected: all existing tests pass.

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id
git add bin-tts-manager/models/speaking/webhook.go
git commit -m "NOJIRA-hide-speaking-pod-id

- bin-tts-manager: Add WebhookMessage struct and ConvertWebhookMessage for Speaking model"
```

---

### Task 2: Update servicehandler interface and implementation

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (lines 854-860, interface declarations)
- Modify: `bin-api-manager/pkg/servicehandler/speaking.go` (all public methods)

**Step 1: Update interface in main.go**

In `bin-api-manager/pkg/servicehandler/main.go`, change lines 854-860 from:

```go
	SpeakingCreate(ctx context.Context, a *amagent.Agent, referenceType tmstreaming.ReferenceType, referenceID uuid.UUID, language string, provider string, voiceID string, direction tmstreaming.Direction) (*tmspeaking.Speaking, error)
	SpeakingGet(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
	SpeakingList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmspeaking.Speaking, error)
	SpeakingSay(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID, text string) (*tmspeaking.Speaking, error)
	SpeakingFlush(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
	SpeakingStop(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
	SpeakingDelete(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
```

To:

```go
	SpeakingCreate(ctx context.Context, a *amagent.Agent, referenceType tmstreaming.ReferenceType, referenceID uuid.UUID, language string, provider string, voiceID string, direction tmstreaming.Direction) (*tmspeaking.WebhookMessage, error)
	SpeakingGet(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
	SpeakingList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmspeaking.WebhookMessage, error)
	SpeakingSay(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID, text string) (*tmspeaking.WebhookMessage, error)
	SpeakingFlush(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
	SpeakingStop(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
	SpeakingDelete(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
```

**Step 2: Update speaking.go implementation**

Replace the entire file `bin-api-manager/pkg/servicehandler/speaking.go` with:

```go
package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// speakingGet gets a speaking record.
func (h *serviceHandler) speakingGet(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	res, err := h.reqHandler.TTSV1SpeakingGet(ctx, speakingID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SpeakingCreate creates a new speaking session.
func (h *serviceHandler) SpeakingCreate(ctx context.Context, a *amagent.Agent, referenceType tmstreaming.ReferenceType, referenceID uuid.UUID, language string, provider string, voiceID string, direction tmstreaming.Direction) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingCreate(ctx, a.CustomerID, referenceType, referenceID, language, provider, voiceID, direction)
	if err != nil {
		log.Errorf("Could not create speaking. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", tmp).Debugf("Created speaking. speaking_id: %s", tmp.ID)

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingGet retrieves a speaking session.
func (h *serviceHandler) SpeakingGet(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	tmp, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingList retrieves a list of speaking sessions.
func (h *serviceHandler) SpeakingList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingList",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[tmspeaking.Field]any{
		tmspeaking.FieldCustomerID: a.CustomerID,
		tmspeaking.FieldDeleted:    false,
	}

	tmps, err := h.reqHandler.TTSV1SpeakingGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get speaking list. err: %v", err)
		return nil, err
	}

	res := make([]*tmspeaking.WebhookMessage, len(tmps))
	for i, s := range tmps {
		res[i] = s.ConvertWebhookMessage()
	}

	return res, nil
}

// SpeakingSay sends text to be spoken. Pod-targeted.
func (h *serviceHandler) SpeakingSay(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID, text string) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingSay",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", s).Debugf("Retrieved speaking info. speaking_id: %s", s.ID)

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingSay(ctx, s.PodID, speakingID, text)
	if err != nil {
		log.Errorf("Could not say text. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingFlush flushes pending text from the speaking queue. Pod-targeted.
func (h *serviceHandler) SpeakingFlush(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingFlush",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", s).Debugf("Retrieved speaking info. speaking_id: %s", s.ID)

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingFlush(ctx, s.PodID, speakingID)
	if err != nil {
		log.Errorf("Could not flush speaking. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingStop stops the speaking session. Pod-targeted.
func (h *serviceHandler) SpeakingStop(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingStop",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", s).Debugf("Retrieved speaking info. speaking_id: %s", s.ID)

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingStop(ctx, s.PodID, speakingID)
	if err != nil {
		log.Errorf("Could not stop speaking. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingDelete soft-deletes a speaking session.
// Stops the streaming session first (pod-targeted) before deleting the DB record (shared queue).
func (h *serviceHandler) SpeakingDelete(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// Stop the streaming session first (pod-targeted) to ensure proper cleanup
	if s.Status == tmspeaking.StatusActive || s.Status == tmspeaking.StatusInitiating {
		if _, errStop := h.reqHandler.TTSV1SpeakingStop(ctx, s.PodID, speakingID); errStop != nil {
			log.Errorf("Could not stop speaking before delete. err: %v", errStop)
		}
	}

	tmp, err := h.reqHandler.TTSV1SpeakingDelete(ctx, speakingID)
	if err != nil {
		log.Infof("Could not delete speaking. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}
```

**Key changes:**
- All public method return types change from `*tmspeaking.Speaking` to `*tmspeaking.WebhookMessage` (or `[]*tmspeaking.WebhookMessage` for List)
- Each method calls `.ConvertWebhookMessage()` on the result before returning
- `speakingGet()` (private) stays unchanged — returns `*Speaking` with `PodID` for internal routing
- `SpeakingList` converts each item via a loop

**Step 3: Regenerate the servicehandler mock**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go generate ./pkg/servicehandler/...
```
Expected: `mock_main.go` regenerated with updated Speaking method signatures.

**Step 4: Verify bin-api-manager builds**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go build ./cmd/...
```
Expected: clean build, no errors.

**Step 5: Run bin-api-manager tests**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go test ./...
```
Expected: all tests pass.

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id
git add bin-api-manager/pkg/servicehandler/main.go bin-api-manager/pkg/servicehandler/speaking.go bin-api-manager/pkg/servicehandler/mock_main.go
git commit -m "NOJIRA-hide-speaking-pod-id

- bin-api-manager: Change speaking servicehandler to return WebhookMessage instead of raw Speaking struct
- bin-api-manager: Regenerate servicehandler mock with updated signatures"
```

---

### Task 3: Remove pod_id from OpenAPI spec and regenerate

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (lines 5494-5497)
- Regenerate: `bin-openapi-manager/gens/models/gen.go`
- Regenerate: `bin-api-manager/gens/openapi_server/gen.go`

**Step 1: Remove pod_id from OpenAPI schema**

In `bin-openapi-manager/openapi/openapi.yaml`, delete lines 5494-5497 (the `pod_id` property under `TtsManagerSpeaking`):

```yaml
        pod_id:
          type: string
          description: Kubernetes pod hosting this session
          example: "tts-manager-7b8f9c0d-pod1"
```

**Step 2: Regenerate bin-openapi-manager models**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-openapi-manager
go generate ./...
```
Expected: `gens/models/gen.go` regenerated without the `PodId` field in `TtsManagerSpeaking`.

**Step 3: Vendor and regenerate bin-api-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```
Expected: `gens/openapi_server/gen.go` regenerated.

**Step 4: Verify both services build**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-openapi-manager
go build ./...

cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go build ./cmd/...
```
Expected: both build cleanly.

**Step 5: Run tests for both services**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-openapi-manager
go test ./...

cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go test ./...
```
Expected: all tests pass.

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id
git add bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/gens/models/gen.go bin-api-manager/gens/openapi_server/gen.go
git commit -m "NOJIRA-hide-speaking-pod-id

- bin-openapi-manager: Remove pod_id from TtsManagerSpeaking schema
- bin-openapi-manager: Regenerate OpenAPI models
- bin-api-manager: Regenerate OpenAPI server code"
```

---

### Task 4: Add WebhookMessage rule to CLAUDE.md

**Files:**
- Modify: `CLAUDE.md` (root, after "Debug Logging for Retrieved Data" section, before "Common Gotchas")

**Step 1: Add the rule**

In root `CLAUDE.md`, insert the following new section after the "Debug Logging for Retrieved Data" section (after line 459) and before "### Common Gotchas" (line 461):

```markdown
### WebhookMessage Pattern for External API Responses (MANDATORY)

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

**When adding a new API resource:**
- Create `webhook.go` alongside the model definition
- Omit any fields that are infrastructure-specific or internal-only
- Update the OpenAPI schema to match `WebhookMessage` fields (not the internal struct)

```

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id
git add CLAUDE.md
git commit -m "NOJIRA-hide-speaking-pod-id

- docs: Add WebhookMessage pattern rule to CLAUDE.md for external API responses"
```

---

### Task 5: Full verification and lint

**Step 1: Run full verification for bin-tts-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-tts-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all steps pass.

**Step 2: Run full verification for bin-api-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all steps pass.

**Step 3: Run full verification for bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all steps pass.

**Step 4: If lint or tests fail, fix issues and amend last commit or create new commit**

**Step 5: Push and create PR**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-hide-speaking-pod-id
git push -u origin NOJIRA-hide-speaking-pod-id
```

Then create PR with:
- Title: `NOJIRA-hide-speaking-pod-id`
- Body: Summary of changes with project prefixes
