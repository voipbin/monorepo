# Add Queue direct_hash Support — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `direct_hash` support to the queue resource so callers can dial `direct.<hash>@sip.voipbin.net` to enter a queue directly.

**Architecture:** Mirror the existing direct_hash pattern used by agent, conference, extension, AI, and AI team. Auto-create hash on queue creation. Add call-manager dispatch for `"queue"` resource type using `queue_join` flow action.

**Tech Stack:** Go, MySQL (Alembic migration), RabbitMQ RPC, OpenAPI

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support`

---

### Task 1: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<auto>_add_queue_direct_hash.py` (via `alembic revision`)

**Step 1: Generate migration file**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-dbscheme-manager/bin-manager/main
alembic -c alembic.ini revision -m "add_queue_direct_hash"
```

**Step 2: Edit the generated migration file**

```python
def upgrade():
    op.execute("ALTER TABLE queue_queues ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255)")


def downgrade():
    op.execute("ALTER TABLE queue_queues DROP COLUMN direct_id, DROP COLUMN direct_hash")
```

**Step 3: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-dbscheme-manager: Add direct_id and direct_hash columns to queue_queues table
```

---

### Task 2: Queue Model Updates

**Files:**
- Modify: `bin-queue-manager/models/queue/queue.go:12-42`
- Modify: `bin-queue-manager/models/queue/field.go:6-35`
- Modify: `bin-queue-manager/models/queue/webhook.go:13-67`

**Step 1: Add DirectID and DirectHash fields to Queue struct**

In `queue.go`, add after the `TagIDs` field (line 21):

```go
	// direct hash
	DirectID   uuid.UUID `json:"direct_id,omitempty" db:"direct_id,uuid"`    // direct id for direct hash
	DirectHash string    `json:"direct_hash,omitempty" db:"direct_hash"`     // direct hash
```

**Step 2: Add Field constants**

In `field.go`, add after `FieldTagIDs` (line 14):

```go
	FieldDirectID   Field = "direct_id"   // direct_id
	FieldDirectHash Field = "direct_hash" // direct_hash
```

**Step 3: Add DirectHash to WebhookMessage**

In `webhook.go`, add after `TagIDs` field (line 22):

```go
	// direct hash
	DirectHash string `json:"direct_hash,omitempty"` // direct hash
```

**Step 4: Update ConvertWebhookMessage()**

In `webhook.go`, add `DirectHash: h.DirectHash,` to the conversion method, after `TagIDs: h.TagIDs,` (line 50):

```go
		DirectHash: h.DirectHash,
```

**Step 5: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-queue-manager: Add DirectID and DirectHash fields to Queue model
- bin-queue-manager: Add FieldDirectID and FieldDirectHash constants
- bin-queue-manager: Add DirectHash to WebhookMessage
```

---

### Task 3: Queue Handler — DirectHashRegenerate

**Files:**
- Create: `bin-queue-manager/pkg/queuehandler/direct_hash.go`
- Modify: `bin-queue-manager/pkg/queuehandler/main.go:28-68` (add interface method)

**Step 1: Add DirectHashRegenerate to QueueHandler interface**

In `main.go`, add before `EventCUCustomerDeleted` (line 67):

```go
	DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
```

**Step 2: Create direct_hash.go**

```go
package queuehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given queue.
func (h *queueHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "DirectHashRegenerate",
		"queue_id": id,
	})

	// get current queue
	q, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, fmt.Errorf("could not get queue: %w", err)
	}
	log.WithField("queue", q).Debugf("Retrieved queue info. queue_id: %s", q.ID)

	// regenerate or create direct
	var directID uuid.UUID
	var directHash string
	if q.DirectID != uuid.Nil {
		d, err := h.reqHandler.DirectV1DirectRegenerate(ctx, q.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	} else {
		d, err := h.reqHandler.DirectV1DirectCreate(ctx, q.CustomerID, "queue", id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash created. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	}

	// update queue with new direct info
	fields := map[queue.Field]any{
		queue.FieldDirectID:   directID,
		queue.FieldDirectHash: directHash,
	}
	if err := h.db.QueueUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update queue direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update queue: %w", err)
	}

	// return updated queue
	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-queue-manager: Add DirectHashRegenerate method to QueueHandler interface
- bin-queue-manager: Implement DirectHashRegenerate following agent/conference pattern
```

---

### Task 4: Auto-Create direct_hash on Queue Creation

**Files:**
- Modify: `bin-queue-manager/pkg/queuehandler/create.go:19-101`

**Step 1: Add direct hash creation after queue ID generation**

In `create.go`, after the queue ID is generated (line 55-56) and before the routing method validation (line 58), add:

```go
	// create direct hash
	d, err := h.reqHandler.DirectV1DirectCreate(ctx, customerID, "queue", id)
	if err != nil {
		log.Errorf("Could not create direct hash. err: %v", err)
		return nil, fmt.Errorf("could not create direct hash: %w", err)
	}
	log.WithField("direct", d).Debugf("Created direct hash. direct_id: %s", d.ID)
```

**Step 2: Add DirectID and DirectHash to the queue struct initialization**

In the queue struct literal (lines 63-86), add after `TagIDs: tagIDs,` (line 73):

```go
		DirectID:   d.ID,
		DirectHash: d.Hash,
```

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-queue-manager: Auto-create direct hash during queue creation
```

---

### Task 5: Listen Handler — Direct Hash Regenerate Endpoint

**Files:**
- Create: `bin-queue-manager/pkg/listenhandler/v1_queues_direct_hash.go`
- Modify: `bin-queue-manager/pkg/listenhandler/main.go:41-71` (add regex + route case)

**Step 1: Add regex pattern**

In `main.go`, add after `reqV1QueuesIDExecuteRun` (line 55):

```go
	reqV1QueuesIDDirectHashRegenerate = regexp.MustCompile("/v1/queues/" + regUUID + "/direct-hash-regenerate$")
```

**Step 2: Add route case in processRequest switch**

In `main.go`, add before the `// queuecalls` section comment (line 218), after the execute_run case (line 216):

```go
	// PUT /queues/<queue-id>/direct-hash-regenerate
	case reqV1QueuesIDDirectHashRegenerate.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1QueuesIDDirectHashRegeneratePut(ctx, m)
		requestType = "/v1/queues/<queue-id>/direct-hash-regenerate"
```

**Step 3: Create v1_queues_direct_hash.go**

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

// processV1QueuesIDDirectHashRegeneratePut handles PUT /v1/queues/<queue-id>/direct-hash-regenerate request
func (h *listenHandler) processV1QueuesIDDirectHashRegeneratePut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1QueuesIDDirectHashRegeneratePut",
		"queue_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.queueHandler.DirectHashRegenerate(ctx, id)
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

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-queue-manager: Add PUT /v1/queues/{id}/direct-hash-regenerate endpoint
- bin-queue-manager: Add regex route and handler for direct hash regeneration
```

---

### Task 6: Call Manager — Queue Direct Hash Routing

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go:96-112` (add queue case)
- Create: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_direct_queue.go`

**Step 1: Add "queue" case to dispatch switch**

In `start_incoming_domain_type_sip.go`, add before the `default` case (line 108):

```go
	case "queue":
		return h.startIncomingDomainTypeSIPDirectQueue(ctx, cn, d, source)
```

**Step 2: Create start_incoming_domain_type_sip_direct_queue.go**

```go
package callhandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// startIncomingDomainTypeSIPDirectQueue handles direct hash call routed to a queue resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectQueue(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectQueue",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	q, err := h.reqHandler.QueueV1QueueGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("queue", q).Debugf("Retrieved queue info. queue_id: %s", q.ID)

	destination := &commonaddress.Address{
		Type:       commonaddress.TypeQueue,
		Target:     q.ID.String(),
		TargetName: q.Name,
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeQueueJoin,
			Option: fmaction.ConvertOption(fmaction.OptionQueueJoin{
				QueueID: q.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, q.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct queue join. queue_id: %s", q.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, q.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}
```

**Note:** Check that `commonaddress.TypeQueue` exists. If it doesn't, use `commonaddress.TypeLine` or the appropriate type, or check how queue_join actions set the destination type in existing flow execution code. Search for `TypeQueue` in `bin-common-handler/models/address/` to verify.

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-call-manager: Add "queue" case to SIP direct hash dispatch
- bin-call-manager: Add queue direct hash routing handler with queue_join flow action
```

---

### Task 7: OpenAPI Schema Update

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (QueueManagerQueue schema)
- Create: `bin-openapi-manager/openapi/paths/queues/id_direct_hash_regenerate.yaml`

**Step 1: Read bin-openapi-manager/CLAUDE.md for AI-Native spec rules**

**Step 2: Add direct_hash field to QueueManagerQueue schema**

In `openapi.yaml`, in the `QueueManagerQueue` schema properties, add after `tag_ids`:

```yaml
    direct_hash:
      type: string
      description: "Hash for direct access via SIP URI sip:direct.<hash>@sip.voipbin.net. Returned from the resource's `direct_hash` field."
      example: "a8f3b2c1d4e5"
```

**Step 3: Create id_direct_hash_regenerate.yaml**

Use the agent's `id_direct_hash_regenerate.yaml` as template, replacing `Agent`/`agent` with `Queue`/`queue`:

```yaml
post:
  summary: Regenerate direct hash for queue
  description: Regenerates the direct hash for the specified queue. If no direct hash exists, one is created. Returns the updated queue with the new direct_hash.
  tags:
    - Queue
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
        example: "550e8400-e29b-41d4-a716-446655440000"
      description: "The unique identifier of the queue. Returned from the `GET /queues` response."
  responses:
    '200':
      description: Direct hash regenerated successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/QueueManagerQueue'
    '400':
      description: Unable to regenerate direct hash.
```

**Step 4: Add path reference in openapi.yaml**

Add in the paths section near other queue endpoints:

```yaml
  /queues/{id}/direct-hash-regenerate:
    $ref: './paths/queues/id_direct_hash_regenerate.yaml'
```

**Step 5: Run verification for openapi-manager and api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```
NOJIRA-Add-queue-direct-hash-support

- bin-openapi-manager: Add direct_hash field to QueueManagerQueue schema
- bin-openapi-manager: Add /queues/{id}/direct-hash-regenerate endpoint spec
- bin-api-manager: Regenerate server code from updated OpenAPI spec
```

---

### Task 8: Final Verification and PR

**Step 1: Run full verification for all changed services**

```bash
# queue-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# call-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-queue-direct-hash-support
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Add-queue-direct-hash-support
```

PR title: `NOJIRA-Add-queue-direct-hash-support`
