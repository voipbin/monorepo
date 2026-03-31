# Add Flow Direct Hash Support — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add direct hash support to flows so callers can dial `sip:direct.<hash>@sip.voipbin.net` to trigger a flow's action sequence directly.

**Architecture:** Mirror the queue direct hash pattern: auto-create hash on flow creation (TypeFlow + Persist=true only), add regeneration endpoint, add call-manager routing that delegates to `startCallTypeFlow()` directly without prepending any actions.

**Tech Stack:** Go, MySQL (Alembic migrations), RabbitMQ RPC, Squirrel query builder, OpenAPI/Sphinx RST docs.

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support`

---

### Task 1: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<auto>_flow_flows_add_direct_id_and_direct_hash.py`

**Step 1: Create migration file**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-dbscheme-manager/bin-manager/main
alembic -c alembic.ini revision -m "flow_flows_add_direct_id_and_direct_hash"
```

**Step 2: Edit the generated migration file**

Add to `upgrade()`:
```python
def upgrade():
    op.execute("ALTER TABLE flow_flows ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255)")
```

Add to `downgrade()`:
```python
def downgrade():
    op.execute("ALTER TABLE flow_flows DROP COLUMN direct_id, DROP COLUMN direct_hash")
```

Reference: `bin-dbscheme-manager/bin-manager/main/versions/3ee2e81bc85c_queue_queues_add_direct_id_and_direct_hash.py` (queue migration for identical pattern).

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-dbscheme-manager: Add Alembic migration for direct_id and direct_hash columns on flow_flows"
```

---

### Task 2: Flow Model Changes

**Files:**
- Modify: `bin-flow-manager/models/flow/flow.go:27` (add fields before `OnCompleteFlowID`)
- Modify: `bin-flow-manager/models/flow/field.go:18` (add field constants before `FieldOnCompleteFlowID`)
- Modify: `bin-flow-manager/models/flow/webhook.go:24` (add to WebhookMessage and converter)

**Step 1: Add fields to Flow struct**

In `flow.go`, add after the `Actions` field (line 25) and before `OnCompleteFlowID` (line 27):
```go
	DirectID   uuid.UUID `json:"direct_id,omitempty" db:"direct_id,uuid"`
	DirectHash string    `json:"direct_hash,omitempty" db:"direct_hash"`
```

**Step 2: Add field constants**

In `field.go`, add after `FieldActions` (line 16):
```go
	FieldDirectID   Field = "direct_id"   // direct_id
	FieldDirectHash Field = "direct_hash" // direct_hash
```

**Step 3: Add to WebhookMessage**

In `webhook.go`, add `DirectHash` to the struct (after `Actions`, before `OnCompleteFlowID`):
```go
	DirectHash string `json:"direct_hash,omitempty"`
```

Update `ConvertWebhookMessage()` to include:
```go
		DirectHash: h.DirectHash,
```

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-flow-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-flow-manager: Add DirectID and DirectHash fields to Flow model
- bin-flow-manager: Add DirectHash to WebhookMessage for external API exposure
- bin-flow-manager: Add FieldDirectID and FieldDirectHash field constants"
```

---

### Task 3: Flow Handler — DirectHashRegenerate + Create + Delete

**Files:**
- Create: `bin-flow-manager/pkg/flowhandler/direct_hash.go`
- Modify: `bin-flow-manager/pkg/flowhandler/main.go:46` (add interface method after `Delete`)
- Modify: `bin-flow-manager/pkg/flowhandler/db.go:90-131` (Create — add direct hash auto-creation)
- Modify: `bin-flow-manager/pkg/flowhandler/db.go:190-210` (Delete — add direct hash cleanup)

**Step 1: Create `direct_hash.go`**

Follow the queue pattern exactly (`bin-queue-manager/pkg/queuehandler/direct_hash.go`):

```go
package flowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/flow"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given flow.
func (h *flowHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DirectHashRegenerate",
		"flow_id": id,
	})

	// get current flow
	f, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, fmt.Errorf("could not get flow: %w", err)
	}
	log.WithField("flow", f).Debugf("Retrieved flow info. flow_id: %s", f.ID)

	// regenerate or create direct
	var directID uuid.UUID
	var directHash string
	if f.DirectID != uuid.Nil {
		d, err := h.reqHandler.DirectV1DirectRegenerate(ctx, f.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	} else {
		d, err := h.reqHandler.DirectV1DirectCreate(ctx, f.CustomerID, "flow", id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash created. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	}

	// update flow with new direct info
	fields := map[flow.Field]any{
		flow.FieldDirectID:   directID,
		flow.FieldDirectHash: directHash,
	}
	if err := h.db.FlowUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update flow direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update flow: %w", err)
	}

	// return updated flow
	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 2: Add `DirectHashRegenerate` to FlowHandler interface**

In `main.go`, add after the `Delete` method (line 46):
```go
	DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
```

**Step 3: Modify `Create` in `db.go` for auto-creation**

In the `Create` method, after generating the ID (line 90) and before building the flow struct (line 91), add direct hash creation for `TypeFlow + Persist`:

```go
	id := h.util.UUIDCreate()

	// create direct hash for persistent TypeFlow flows
	var directID uuid.UUID
	var directHash string
	if flowType == flow.TypeFlow && persist {
		d, errDirect := h.reqHandler.DirectV1DirectCreate(ctx, customerID, "flow", id)
		if errDirect != nil {
			log.Errorf("Could not create direct hash. err: %v", errDirect)
			return nil, fmt.Errorf("could not create direct hash: %w", errDirect)
		}
		log.WithField("direct", d).Debugf("Created direct hash. direct_id: %s", d.ID)
		directID = d.ID
		directHash = d.Hash
	}

	f := &flow.Flow{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type: flowType,

		Name:   name,
		Detail: detail,

		Persist: persist,

		Actions: a,

		DirectID:   directID,
		DirectHash: directHash,

		OnCompleteFlowID: onCompleteFlowID,

		TMCreate: h.util.TimeNow(),
		TMUpdate: nil,
		TMDelete: nil,
	}
```

After the persist DB create (line 114-116), add cleanup on failure:
```go
	case f.Persist:
		if err := h.db.FlowCreate(ctx, f); err != nil {
			log.Errorf("Could not create the flow in the database. err: %v", err)
			// cleanup orphaned direct hash
			if directID != uuid.Nil {
				if _, errDelete := h.reqHandler.DirectV1DirectDelete(ctx, directID); errDelete != nil {
					log.Errorf("Could not cleanup orphaned direct. direct_id: %s, err: %v", directID, errDelete)
				}
			}
			return nil, err
		}
```

**Step 4: Modify `Delete` in `db.go` for cleanup**

In the `Delete` method, before `h.db.FlowDelete(ctx, id)` (line 198), add:

```go
	// get flow to check for direct hash cleanup
	f, errGet := h.db.FlowGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get flow for direct hash cleanup. err: %v", errGet)
		return nil, errGet
	}

	// delete direct hash via direct-manager (best-effort, don't block flow deletion)
	if f.DirectID != uuid.Nil {
		if _, errDirect := h.reqHandler.DirectV1DirectDelete(ctx, f.DirectID); errDirect != nil {
			log.Errorf("Could not delete direct hash. direct_id: %s, err: %v", f.DirectID, errDirect)
		}
	}

	err := h.db.FlowDelete(ctx, id)
```

Remove the duplicate `err` declaration since we use `:=` above now.

**Step 5: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-flow-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-flow-manager: Add DirectHashRegenerate method to flowhandler
- bin-flow-manager: Auto-create direct hash on flow creation for TypeFlow with Persist=true
- bin-flow-manager: Add best-effort direct hash cleanup on flow deletion"
```

---

### Task 4: Flow Listen Handler — Regenerate Endpoint

**Files:**
- Create: `bin-flow-manager/pkg/listenhandler/v1_flows_direct_hash.go`
- Modify: `bin-flow-manager/pkg/listenhandler/main.go:66` (add regex pattern after `regV1FlowsIDActionsID`)
- Modify: `bin-flow-manager/pkg/listenhandler/main.go:262-269` (add route to processRequest switch, before the `regV1FlowsIDActions` case)

**Step 1: Create `v1_flows_direct_hash.go`**

Follow the queue pattern (`bin-queue-manager/pkg/listenhandler/v1_queues_direct_hash.go`):

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1FlowsIDDirectHashRegeneratePost handles POST /v1/flows/<flow-id>/direct-hash-regenerate request
func (h *listenHandler) processV1FlowsIDDirectHashRegeneratePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1FlowsIDDirectHashRegeneratePost",
		"flow_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.flowHandler.DirectHashRegenerate(ctx, id)
	if err != nil {
		log.Errorf("Could not regenerate direct hash. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
```

**Step 2: Add regex pattern in `main.go`**

After `regV1FlowsIDActionsID` (line 66), add:
```go
	regV1FlowsIDDirectHashRegenerate = regexp.MustCompile("/v1/flows/" + regUUID + "/direct-hash-regenerate$")
```

**Step 3: Add route in `processRequest` switch**

Add BEFORE the existing `regV1FlowsIDActions` case (line 262-265). It must come before `regV1FlowsIDActions` since that pattern would also match the `/direct-hash-regenerate` suffix:

```go
	// flows/<flow-id>/direct-hash-regenerate
	case regV1FlowsIDDirectHashRegenerate.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/flows/<flow-id>/direct-hash-regenerate"
		response, err = h.processV1FlowsIDDirectHashRegeneratePost(ctx, m)
```

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-flow-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-flow-manager: Add POST /v1/flows/{id}/direct-hash-regenerate listen handler endpoint
- bin-flow-manager: Register direct-hash-regenerate route in processRequest switch"
```

---

### Task 5: RequestHandler — Flow Direct Hash Regenerate RPC

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/flow_flow.go:218` (add new method at end)
- Modify: `bin-common-handler/pkg/requesthandler/main.go:979` (add to RequestHandler interface after `FlowV1FlowCountByCustomerID`)

**Step 1: Add RPC method to `flow_flow.go`**

Add at the end of the file:
```go
// FlowV1FlowDirectHashRegenerate sends a request to flow-manager
// to regenerate (or create) the direct hash for the given flow.
func (r *requestHandler) FlowV1FlowDirectHashRegenerate(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s/direct-hash-regenerate", flowID)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/flows/<flow-id>/direct-hash-regenerate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 2: Add to RequestHandler interface in `main.go`**

After `FlowV1FlowCountByCustomerID` (line 979), add:
```go
	FlowV1FlowDirectHashRegenerate(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
```

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-common-handler: Add FlowV1FlowDirectHashRegenerate RPC method to requesthandler
- bin-common-handler: Add method to RequestHandler interface"
```

---

### Task 6: Call Manager — Direct Flow Routing

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go:109` (add "flow" case before `default`)
- Create: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_direct_flow.go`

**Step 1: Create `start_incoming_domain_type_sip_direct_flow.go`**

Unlike other resources, the flow handler does NOT create a temporary flow. It uses the flow directly:

```go
package callhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// startIncomingDomainTypeSIPDirectFlow handles direct hash call routed to a flow resource.
// Unlike other resource types, the flow already defines the complete action sequence,
// so no temporary flow is created — it delegates directly to startCallTypeFlow.
func (h *callHandler) startIncomingDomainTypeSIPDirectFlow(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectFlow",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	// get flow info to validate it exists and get customer_id
	f, err := h.reqHandler.FlowV1FlowGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("flow", f).Debugf("Retrieved flow info. flow_id: %s", f.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: d.ResourceID.String(),
	}

	h.startCallTypeFlow(ctx, cn, f.CustomerID, f.ID, source, destination, nil)
	return nil
}
```

**Step 2: Add "flow" case to dispatch switch**

In `start_incoming_domain_type_sip.go`, add before the `default` case (line 110):
```go
	case "flow":
		return h.startIncomingDomainTypeSIPDirectFlow(ctx, cn, d, source)
```

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-call-manager: Add flow case to direct hash routing dispatch
- bin-call-manager: Add startIncomingDomainTypeSIPDirectFlow handler that delegates directly to startCallTypeFlow"
```

---

### Task 7: OpenAPI Schema Update

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add `direct_hash` to `FlowManagerFlow` schema)

**Step 1: Add `direct_hash` field to FlowManagerFlow schema**

In `openapi.yaml`, find the `FlowManagerFlow` schema (around line 4600). Add the `direct_hash` property after `actions` and before `on_complete_flow_id`:

```yaml
        direct_hash:
          type: string
          description: "Hash for direct access via SIP URI sip:direct.<hash>@sip.voipbin.net. Returned from the resource's `direct_hash` field."
          example: "direct.a8f3b2c1d4e5"
```

**Step 2: Regenerate OpenAPI types**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Regenerate API manager server code**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-openapi-manager: Add direct_hash field to FlowManagerFlow schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 8: RST Documentation Update

**Files:**
- Modify: `bin-api-manager/docsdev/source/direct_hash_overview.rst` (add flow to resource table and descriptions)
- Modify: `bin-api-manager/docsdev/source/flow_struct_flow.rst` (add direct_hash field)

**Step 1: Update direct_hash_overview.rst**

In the Supported Resources table (around line 83-97), add a row for Flow:
```rst
+---------------+----------------+---------------------------------------------------+-------------------------------------------+
| Flow          | Yes            | ``POST /flows/{id}/direct-hash-regenerate``       | :ref:`flow-overview`                      |
+---------------+----------------+---------------------------------------------------+-------------------------------------------+
```

Update line 14 to include "flows":
```
Seven resource types support direct hash: **extensions**, **agents**, **conferences**, **queues**, **flows**, **AIs**, and **teams**.
```

In the routing flow section (lines 57-63), add:
```
- **Flow**: The activeflow executes the flow's defined action sequence as-is.
```

In the Managing Direct Hashes section (line 107), add "flows" to the auto-created list:
```
For extensions, conferences, teams, queues, and flows, a direct hash is generated automatically when the resource is created.
```

Add a flow-specific use case (line 139):
```
- **Flow testing**: Share a direct hash SIP URI for a flow to allow testing the flow's action sequence without configuring an inbound number.
```

**Step 2: Update flow_struct_flow.rst**

Add `direct_hash` field description to the struct documentation.

**Step 3: Rebuild HTML docs**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support
git add bin-api-manager/docsdev/source/
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-api-manager: Update direct_hash_overview.rst to include flow as supported resource
- bin-api-manager: Add direct_hash field to flow_struct_flow.rst
- bin-api-manager: Rebuild HTML documentation"
```

---

### Task 9: API Manager — ServiceHandler for Flow Direct Hash Regenerate

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/` (add FlowDirectHashRegenerate method)
- Modify: `bin-api-manager/server/flows.go` (add endpoint handler)

**Step 1: Identify existing flow servicehandler pattern**

Check `bin-api-manager/pkg/servicehandler/flow.go` for existing flow methods. Add `FlowDirectHashRegenerate` following the same pattern as other resources (e.g., `QueueDirectHashRegenerate` in `queue.go`).

The method should:
1. Get the flow via the private helper to check permission
2. Call `h.reqHandler.FlowV1FlowDirectHashRegenerate(ctx, flowID)`
3. Return `ConvertWebhookMessage()` result

**Step 2: Add endpoint handler in server**

Check `bin-api-manager/server/flows.go` for existing flow routes. Add the direct-hash-regenerate POST endpoint following the queue's pattern in `server/queues.go`.

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Add-flow-direct-hash-support

- bin-api-manager: Add FlowDirectHashRegenerate servicehandler method
- bin-api-manager: Add POST /flows/{id}/direct-hash-regenerate endpoint handler"
```

---

### Task 10: Final Verification

**Step 1: Run verification on all changed services**

```bash
# bin-flow-manager (main changes)
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-call-manager (routing change)
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-common-handler (new RPC method)
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-flow-direct-hash-support
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Add-flow-direct-hash-support
```
