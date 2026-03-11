# Number Metadata Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `Metadata` JSON field to the Number model with an `rtp_debug` flag, updatable via a dedicated `PUT /numbers/{id}/metadata` endpoint. Enable inclusive OR logic for RTP debug: if either the customer or the number has `rtp_debug: true`, RTP capture is enabled for incoming PSTN and virtual number calls.

**Architecture:** New `Metadata` struct stored as JSON column in `number_numbers`. Exposed via WebhookMessage (unlike customer metadata which is admin-only). Write via new `PUT /v1/numbers/{id}/metadata` endpoint (CustomerAdmin|CustomerManager permission). In call-manager, pass the already-fetched `*nmnumber.Number` through `startCallTypeFlow` so the RTP debug check can OR customer-level and number-level flags. Only PSTN and SIP (virtual) incoming paths pass the number; all other callers pass `nil`. Outgoing calls continue to check only the customer-level flag.

**Tech Stack:** Go, MySQL (JSON column), Alembic (migration), Squirrel (query builder), RabbitMQ RPC, OpenAPI + oapi-codegen

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-number-metadata/`

**All file paths below are relative to the worktree root.**

---

## Design Summary

### Number Metadata Model

```go
// bin-number-manager/models/number/metadata.go
package number

type MetadataKey = string

const (
    MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)

type Metadata struct {
    RTPDebug bool `json:"rtp_debug"`
}
```

### RTP Debug Inclusive OR Logic

For incoming PSTN and SIP (virtual number) calls, the RTP debug check becomes:

```go
rtpDebugEnabled := cs.Metadata.RTPDebug
if num != nil {
    rtpDebugEnabled = rtpDebugEnabled || num.Metadata.RTPDebug
}
if rtpDebugEnabled {
    // enable RTP capture
}
```

### Affected Services

| Service | Changes |
|---------|---------|
| bin-dbscheme-manager | Alembic migration: add `metadata` JSON column to `number_numbers` |
| bin-number-manager | Metadata type, field const, struct update, webhook update, UpdateMetadata handler, listen route, request struct |
| bin-common-handler | `NumberV1NumberUpdateMetadata` RPC method + interface update |
| bin-api-manager | `NumberUpdateMetadata` servicehandler method + interface update |
| bin-openapi-manager | `NumberManagerMetadata` schema, path file, metadata field on `NumberManagerNumber` |
| bin-call-manager | Pass `*nmnumber.Number` through `startCallTypeFlow`, inclusive OR for PSTN/SIP calls |
| bin-api-manager (RST) | Update struct, tutorial, overview RST docs + rebuild HTML |

---

### Task 1: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_number_numbers_add_column_metadata.py`

**Step 1: Create the Alembic migration file**

Manually create the migration file. The current head revision is `9dddf595c42f`.

```python
"""number_numbers add column metadata

Revision ID: a1b2c3d4e5f6
Revises: 9dddf595c42f
Create Date: 2026-03-12 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = '9dddf595c42f'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE number_numbers ADD metadata JSON DEFAULT NULL AFTER emergency_enabled;""")


def downgrade():
    op.execute("""ALTER TABLE number_numbers DROP COLUMN metadata;""")
```

**Step 2: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-dbscheme-manager: Add metadata JSON column to number_numbers table"
```

---

### Task 2: Number Model — Metadata Type, Field Constant, Struct Update, Webhook Update

**Files:**
- Create: `bin-number-manager/models/number/metadata.go`
- Modify: `bin-number-manager/models/number/number.go` (add Metadata field)
- Modify: `bin-number-manager/models/number/field.go` (add FieldMetadata)
- Modify: `bin-number-manager/models/number/webhook.go` (add Metadata to WebhookMessage + ConvertWebhookMessage)

**Step 1: Create `metadata.go`**

```go
package number

// MetadataKey defines typed keys for number metadata fields.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug enables RTPEngine RTP capture (PCAP) for calls to this number.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)

// Metadata holds configuration flags for a number.
// Can be updated via PUT /numbers/{id}/metadata.
type Metadata struct {
	RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
```

**Step 2: Add `Metadata` field to `Number` struct**

In `number.go`, add after `EmergencyEnabled`:

```go
	EmergencyEnabled bool `json:"emergency_enabled" db:"emergency_enabled"`

	Metadata Metadata `json:"metadata" db:"metadata,json"`
```

**Step 3: Add `FieldMetadata` constant**

In `field.go`, add after `FieldEmergencyEnabled`:

```go
	FieldEmergencyEnabled Field = "emergency_enabled"

	FieldMetadata Field = "metadata"
```

**Step 4: Add `Metadata` to `WebhookMessage` and `ConvertWebhookMessage`**

In `webhook.go`, add after `EmergencyEnabled`:

```go
	EmergencyEnabled bool `json:"emergency_enabled"`

	Metadata Metadata `json:"metadata"`
```

And in `ConvertWebhookMessage()`, add after `EmergencyEnabled`:

```go
		EmergencyEnabled: h.EmergencyEnabled,

		Metadata: h.Metadata,
```

**Step 5: Run verification**

```bash
cd bin-number-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-number-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-number-manager: Add Metadata type, field constant, and struct field to Number model
- bin-number-manager: Include metadata in WebhookMessage for external API responses"
```

---

### Task 3: Number Handler — UpdateMetadata Method

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/main.go` (add UpdateMetadata to interface)
- Modify: `bin-number-manager/pkg/numberhandler/number.go` (add UpdateMetadata implementation)

**Step 1: Add `UpdateMetadata` to interface**

In `main.go`, add after the `Update` method:

```go
	Update(ctx context.Context, id uuid.UUID, fields map[number.Field]any) (*number.Number, error)
	UpdateMetadata(ctx context.Context, id uuid.UUID, metadata number.Metadata) (*number.Number, error)
```

**Step 2: Add `UpdateMetadata` implementation**

In `number.go`, add after `Update`:

```go
// UpdateMetadata updates the number's metadata.
func (h *numberHandler) UpdateMetadata(ctx context.Context, id uuid.UUID, metadata number.Metadata) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "UpdateMetadata",
		"number_id": id,
		"metadata":  metadata,
	})
	log.Debugf("UpdateMetadata. number_id: %s", id)

	fields := map[number.Field]any{
		number.FieldMetadata: metadata,
	}

	res, err := h.dbUpdate(ctx, id, fields, number.EventTypeNumberUpdated)
	if err != nil {
		log.Errorf("Could not update the number metadata. err: %v", err)
		return nil, errors.Wrap(err, "could not update the number metadata")
	}

	return res, nil
}
```

**Step 3: Run verification**

```bash
cd bin-number-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-number-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-number-manager: Add UpdateMetadata method to NumberHandler interface and implementation"
```

---

### Task 4: Number ListenHandler — Metadata Route and Request Struct

**Files:**
- Modify: `bin-number-manager/pkg/listenhandler/main.go` (add regex + switch case)
- Create: `bin-number-manager/pkg/listenhandler/v1_numbers_metadata.go` (process function)
- Modify: `bin-number-manager/pkg/listenhandler/models/request/v1_numbers.go` (add request struct)

**Step 1: Add request struct**

In `v1_numbers.go`, add at the end:

```go
// V1DataNumbersIDMetadataPut is
// v1 data type request struct for
// /v1/numbers/<id>/metadata PUT
type V1DataNumbersIDMetadataPut struct {
	Metadata number.Metadata `json:"metadata"`
}
```

**Step 2: Add regex in `main.go`**

Add after `regV1NumbersIDFlowIDs`:

```go
	regV1NumbersIDFlowIDs = regexp.MustCompile("/v1/numbers/" + regUUID + "/flow_ids$")
	regV1NumbersIDMetadata = regexp.MustCompile("/v1/numbers/" + regUUID + "/metadata$")
```

**Step 3: Add switch case in `processRequest`**

Add BEFORE the `regV1NumbersID` cases (more specific regex must come first). Add after the `regV1NumbersIDFlowIDs` PUT case:

```go
	// PUT /numbers/<id>/metadata
	case regV1NumbersIDMetadata.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumbersIDMetadataPut(ctx, m)
		requestType = "/v1/numbers/<number-id>/metadata"
```

**Step 4: Create `v1_numbers_metadata.go`**

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/pkg/listenhandler/models/request"
)

// processV1NumbersIDMetadataPut handles PUT /v1/numbers/<number-id>/metadata request
func (h *listenHandler) processV1NumbersIDMetadataPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":      "processV1NumbersIDMetadataPut",
		"number_id": id,
	})
	log.Debug("Executing processV1NumbersIDMetadataPut.")

	var req request.V1DataNumbersIDMetadataPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.numberHandler.UpdateMetadata(ctx, id, req.Metadata)
	if err != nil {
		log.Errorf("Could not update the number's metadata. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

**Step 5: Run verification**

```bash
cd bin-number-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-number-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-number-manager: Add PUT /v1/numbers/{id}/metadata listen route and request struct"
```

---

### Task 5: bin-common-handler — RPC Method for Number Metadata Update

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (add to interface, line ~988)
- Modify: `bin-common-handler/pkg/requesthandler/nunmber_number.go` (add implementation)

**Step 1: Add to interface**

In `main.go`, add after `NumberV1NumberUpdateFlowID`:

```go
	NumberV1NumberUpdateFlowID(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.Number, error)
	NumberV1NumberUpdateMetadata(ctx context.Context, id uuid.UUID, metadata nmnumber.Metadata) (*nmnumber.Number, error)
```

**Step 2: Add implementation**

In `nunmber_number.go`, add after `NumberV1NumberUpdateFlowID`:

```go
// NumberV1NumberUpdateMetadata sends a request to the number-manager
// to update a number's metadata.
// Returns updated number info
func (r *requestHandler) NumberV1NumberUpdateMetadata(ctx context.Context, id uuid.UUID, metadata nmnumber.Metadata) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s/metadata", id)

	data := &nmrequest.V1DataNumbersIDMetadataPut{
		Metadata: metadata,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPut, "number/numbers/<number-id>/metadata", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 3: Run verification for bin-common-handler**

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-number-metadata

- bin-common-handler: Add NumberV1NumberUpdateMetadata RPC method to RequestHandler"
```

---

### Task 6: bin-api-manager — NumberUpdateMetadata Servicehandler

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (add to ServiceHandler interface, line ~571)
- Modify: `bin-api-manager/pkg/servicehandler/numbers.go` (add NumberUpdateMetadata)

**Step 1: Add to interface**

In `main.go`, add after `NumberUpdateFlowIDs`:

```go
	NumberUpdateFlowIDs(ctx context.Context, a *amagent.Agent, id, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdateMetadata(ctx context.Context, a *amagent.Agent, id uuid.UUID, metadata nmnumber.Metadata) (*nmnumber.WebhookMessage, error)
```

**Step 2: Add implementation**

In `numbers.go`, add after `NumberUpdateFlowIDs`:

```go
// NumberUpdateMetadata updates the number's metadata.
// Requires CustomerAdmin or CustomerManager permission.
func (h *serviceHandler) NumberUpdateMetadata(ctx context.Context, a *amagent.Agent, id uuid.UUID, metadata nmnumber.Metadata) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "NumberUpdateMetadata",
		"number_id": id,
		"metadata":  metadata,
	})

	// get number
	n, err := h.numberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, n.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// update number metadata
	tmp, err := h.reqHandler.NumberV1NumberUpdateMetadata(ctx, id, metadata)
	if err != nil {
		log.Errorf("Could not update the number metadata. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

**Step 3: Run verification for bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-api-manager: Add NumberUpdateMetadata servicehandler method for PUT /numbers/{id}/metadata"
```

---

### Task 7: OpenAPI Schema — NumberManagerMetadata + Path + Spec Updates

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add NumberManagerMetadata schema + metadata field to NumberManagerNumber + path reference)
- Create: `bin-openapi-manager/openapi/paths/numbers/id_metadata.yaml` (PUT endpoint)

**Step 1: Add `NumberManagerMetadata` schema**

In `openapi.yaml`, add after the `NumberManagerNumber` schema block (after all NumberManager schemas):

```yaml
    NumberManagerMetadata:
      type: object
      description: |
        Configuration flags for a number. Controls platform behavior
        such as RTP packet capture for debugging audio issues on this specific number.
        Updatable by CustomerAdmin or CustomerManager via `PUT /numbers/{id}/metadata`.
      properties:
        rtp_debug:
          type: boolean
          description: |
            When set to `true`, RTPEngine captures RTP traffic as PCAP files for calls to this number.
            This flag is OR'd with the customer-level `rtp_debug` — if either is `true`, capture is enabled.
            Use this to debug audio quality issues on a specific number without enabling capture for all customer calls.
            Default is `false`. Enabling this increases storage usage — disable after debugging.
          example: false
```

**Step 2: Add `metadata` property to `NumberManagerNumber`**

In the `NumberManagerNumber` properties, add after `emergency_enabled`:

```yaml
        emergency_enabled:
          type: boolean
          description: Whether emergency services are enabled for the number.
          example: false
        metadata:
          $ref: '#/components/schemas/NumberManagerMetadata'
```

**Step 3: Create `id_metadata.yaml` path file**

Create `bin-openapi-manager/openapi/paths/numbers/id_metadata.yaml`:

```yaml
put:
  summary: Update a number's metadata.
  description: |
    Updates configuration flags for a number. Requires `CustomerAdmin` or `CustomerManager` permission.
    The response returns the full number object including the updated metadata.
  tags:
    - Number
  parameters:
    - name: id
      in: path
      required: true
      description: "The unique identifier of the number (UUID). Obtained from the `id` field of `GET /numbers`."
      schema:
        type: string
  requestBody:
    required: true
    content:
      application/json:
        schema:
          $ref: '#/components/schemas/NumberManagerMetadata'
  responses:
    '200':
      description: The updated number object with the new metadata applied.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/NumberManagerNumber'
    '400':
      description: |
        Invalid request. Possible causes:
        - The `id` path parameter is not a valid UUID.
        - The request body is not valid JSON or does not match the `NumberManagerMetadata` schema.
    '403':
      description: |
        Permission denied. The authenticated agent does not have `CustomerAdmin` or `CustomerManager` permission.
```

**Step 4: Add path reference in `openapi.yaml`**

In the `paths:` section, add after the `/numbers/{id}` path reference (wherever numbers paths are grouped):

```yaml
  /numbers/{id}/metadata:
    $ref: './paths/numbers/id_metadata.yaml'
```

**Step 5: Regenerate models in bin-openapi-manager**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./...
```

**Step 6: Regenerate server code in bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

**Step 7: Run verification for both**

```bash
cd bin-openapi-manager && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 8: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-openapi-manager: Add NumberManagerMetadata schema and PUT /numbers/{id}/metadata path
- bin-openapi-manager: Add metadata field to NumberManagerNumber schema
- bin-api-manager: Regenerate OpenAPI server code"
```

---

### Task 8: bin-call-manager — Pass Number Through startCallTypeFlow + Inclusive OR

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start.go` (update `startCallTypeFlow` signature + PSTN caller + conference caller + RTP debug check)
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go` (pass `&numb` from SIP handler, `nil` from direct extension)
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_registrar.go` (pass `nil` from all 4 registrar paths)
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_trunk.go` (pass `nil` from trunk path)

**Step 1: Update `startCallTypeFlow` signature**

In `start.go`, change the signature from:

```go
func (h *callHandler) startCallTypeFlow(ctx context.Context, cn *channel.Channel, customerID uuid.UUID, flowID uuid.UUID, source *commonaddress.Address, destination *commonaddress.Address) {
```

to:

```go
func (h *callHandler) startCallTypeFlow(ctx context.Context, cn *channel.Channel, customerID uuid.UUID, flowID uuid.UUID, source *commonaddress.Address, destination *commonaddress.Address, num *nmnumber.Number) {
```

**Step 2: Update PSTN caller**

In `startIncomingDomainTypePSTN` (start.go:553), change:

```go
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination)
```

to:

```go
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination, &numb)
```

**Step 3: Update conference caller**

In `startIncomingDomainTypeConference` (start.go:511), change:

```go
	h.startCallTypeFlow(ctx, cn, cf.CustomerID, tmpFlow.ID, source, destination)
```

to:

```go
	h.startCallTypeFlow(ctx, cn, cf.CustomerID, tmpFlow.ID, source, destination, nil)
```

**Step 4: Update SIP callers**

In `start_incoming_domain_type_sip.go`:

Line 72 (`startIncomingDomainTypeSIP`), change:
```go
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination)
```
to:
```go
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination, &numb)
```

Line 134 (`startIncomingDomainTypeSIPDirectExtension`), change:
```go
	h.startCallTypeFlow(ctx, cn, ext.CustomerID, f.ID, source, destination)
```
to:
```go
	h.startCallTypeFlow(ctx, cn, ext.CustomerID, f.ID, source, destination, nil)
```

**Step 5: Update registrar callers**

In `start_incoming_domain_type_registrar.go`, update all 4 call sites (lines 141, 207, 261, 323):

```go
	h.startCallTypeFlow(ctx, cn, customerID, f.ID, source, destination, nil)
```

**Step 6: Update trunk caller**

In `start_incoming_domain_type_trunk.go` (line 98):

```go
	h.startCallTypeFlow(ctx, cn, customerID, f.ID, source, destination, nil)
```

**Step 7: Update RTP debug check in `startCallTypeFlow`**

In `start.go`, replace the existing RTP debug block (lines 645-664):

```go
	// RTP debug: check if customer has RTP debug enabled
	cs, errCS := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if errCS != nil {
		log.Errorf("Could not get customer for RTP debug check. customer_id: %s, err: %v", customerID, errCS)
	} else {
		log.WithField("customer", cs).Debugf("Retrieved customer for RTP debug check. customer_id: %s", cs.ID)
		if cs.Metadata.RTPDebug {
			if c.Metadata == nil {
				c.Metadata = map[string]interface{}{}
			}
			c.Metadata[call.MetadataKeyRTPDebug] = true
			if errMeta := h.db.CallUpdate(ctx, c.ID, map[call.Field]any{
				call.FieldMetadata: c.Metadata,
			}); errMeta != nil {
				log.Errorf("Could not update call metadata for RTP debug. err: %v", errMeta)
			} else {
				h.rtpDebugStartRecording(ctx, c, cn)
			}
		}
	}
```

with:

```go
	// RTP debug: check if customer or number has RTP debug enabled (inclusive OR)
	cs, errCS := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if errCS != nil {
		log.Errorf("Could not get customer for RTP debug check. customer_id: %s, err: %v", customerID, errCS)
	} else {
		log.WithField("customer", cs).Debugf("Retrieved customer for RTP debug check. customer_id: %s", cs.ID)
		rtpDebugEnabled := cs.Metadata.RTPDebug
		if num != nil {
			rtpDebugEnabled = rtpDebugEnabled || num.Metadata.RTPDebug
		}
		if rtpDebugEnabled {
			if c.Metadata == nil {
				c.Metadata = map[string]interface{}{}
			}
			c.Metadata[call.MetadataKeyRTPDebug] = true
			if errMeta := h.db.CallUpdate(ctx, c.ID, map[call.Field]any{
				call.FieldMetadata: c.Metadata,
			}); errMeta != nil {
				log.Errorf("Could not update call metadata for RTP debug. err: %v", errMeta)
			} else {
				h.rtpDebugStartRecording(ctx, c, cn)
			}
		}
	}
```

**Step 8: Run verification**

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 9: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Add-number-metadata

- bin-call-manager: Pass number object through startCallTypeFlow for PSTN and SIP calls
- bin-call-manager: Add inclusive OR logic for rtp_debug (customer OR number metadata)"
```

---

### Task 9: RST Documentation Updates

**Files:**
- Modify: `bin-api-manager/docsdev/source/number_struct_number.rst`
- Modify: `bin-api-manager/docsdev/source/number_tutorial.rst`
- Modify: `bin-api-manager/docsdev/source/number_overview.rst`

**Step 1: Update `number_struct_number.rst`**

Add `metadata` field to the JSON block after `emergency_enabled`:

```rst
        "emergency_enabled": <boolean>,
        "metadata": {
            "rtp_debug": <boolean>
        },
```

Add field description after `emergency_enabled`:

```rst
* ``metadata`` (Object): Configuration flags for this number. See :ref:`Metadata <number-struct-number-metadata>`.
  * ``rtp_debug`` (Boolean): When ``true``, RTPEngine captures RTP traffic as PCAP files for calls to this number. This flag is OR'd with the customer-level ``rtp_debug`` — if either is ``true``, capture is enabled. Default is ``false``.
```

Add a new Metadata section after the Status section:

```rst
.. _number-struct-number-metadata:

Metadata
--------

The ``metadata`` object contains configuration flags for the number.

================ ======= ===========
Field            Type    Description
================ ======= ===========
rtp_debug        Boolean When ``true``, RTPEngine captures RTP traffic (PCAP) for incoming PSTN and virtual number calls to this number. OR'd with customer-level flag — if either is ``true``, capture is enabled. Default ``false``. Disable after debugging to reduce storage.
================ ======= ===========
```

Update the Example block to include metadata:

```rst
        "metadata": {
            "rtp_debug": false
        },
```

**Step 2: Update `number_tutorial.rst`**

Add a new section after "Delete number":

```rst
Update number metadata
----------------------

Update per-number configuration flags. Requires ``CustomerAdmin`` or ``CustomerManager`` permission.

.. note:: **AI Implementation Hint**

   The ``rtp_debug`` flag enables RTP packet capture (PCAP) for incoming calls to this specific number.
   It is OR'd with the customer-level ``rtp_debug`` flag — if either is ``true``, capture is enabled.
   Use this when you need to debug audio issues on a specific number without enabling capture for all customer calls.
   Disable after debugging to reduce storage usage.

Example

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/numbers/d5532488-0b2d-11eb-b18c-172ab8f2d3d8/metadata?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "rtp_debug": true
    }'

    {
        "id": "d5532488-0b2d-11eb-b18c-172ab8f2d3d8",
        "number": "+16195734778",
        "type": "normal",
        "call_flow_id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
        "message_flow_id": "00000000-0000-0000-0000-000000000000",
        "name": "Support Line",
        "detail": "",
        "status": "active",
        "t38_enabled": false,
        "emergency_enabled": false,
        "metadata": {
            "rtp_debug": true
        },
        "tm_create": "2020-10-11 01:00:00.000001",
        "tm_update": "2026-03-12 10:30:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
```

**Step 3: Update `number_overview.rst`**

In the "With the Number API you can:" list, update the existing "Manage number settings and metadata" line (already there, line 19). No further changes needed.

**Step 4: Rebuild HTML**

```bash
cd bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

**Step 5: Commit**

```bash
git add bin-api-manager/docsdev/source/number_struct_number.rst
git add bin-api-manager/docsdev/source/number_tutorial.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Add-number-metadata

- bin-api-manager: Add metadata field to number struct RST documentation
- bin-api-manager: Add Update number metadata tutorial section
- bin-api-manager: Rebuild HTML documentation"
```

---

### Task 10: Final Verification

**Step 1: Run full verification for all changed services**

```bash
# bin-number-manager
cd bin-number-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-common-handler
cd ../bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ../bin-openapi-manager && go mod tidy && go mod vendor && go generate ./...

# bin-call-manager
cd ../bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify all tests pass**

Expected: All tests pass across all services.

**Step 3: Check for any missed files**

```bash
git status
git diff --stat
```

---

## Summary of All Changes

| Service | File | Change |
|---------|------|--------|
| bin-dbscheme-manager | `bin-manager/main/versions/<hash>_number_numbers_add_column_metadata.py` | New migration file |
| bin-number-manager | `models/number/metadata.go` | New: Metadata type + MetadataKey const |
| bin-number-manager | `models/number/number.go` | Add `Metadata` field with `db:"metadata,json"` tag |
| bin-number-manager | `models/number/field.go` | Add `FieldMetadata` constant |
| bin-number-manager | `models/number/webhook.go` | Add `Metadata` to WebhookMessage + ConvertWebhookMessage |
| bin-number-manager | `pkg/numberhandler/main.go` | Add `UpdateMetadata` to interface |
| bin-number-manager | `pkg/numberhandler/number.go` | Add `UpdateMetadata` implementation |
| bin-number-manager | `pkg/listenhandler/main.go` | Add regex + switch case for metadata route |
| bin-number-manager | `pkg/listenhandler/v1_numbers_metadata.go` | New: processV1NumbersIDMetadataPut |
| bin-number-manager | `pkg/listenhandler/models/request/v1_numbers.go` | Add `V1DataNumbersIDMetadataPut` struct |
| bin-common-handler | `pkg/requesthandler/main.go` | Add `NumberV1NumberUpdateMetadata` to interface |
| bin-common-handler | `pkg/requesthandler/nunmber_number.go` | Add `NumberV1NumberUpdateMetadata` implementation |
| bin-api-manager | `pkg/servicehandler/main.go` | Add `NumberUpdateMetadata` to interface |
| bin-api-manager | `pkg/servicehandler/numbers.go` | Add `NumberUpdateMetadata` implementation |
| bin-openapi-manager | `openapi/openapi.yaml` | Add `NumberManagerMetadata` schema + metadata field + path ref |
| bin-openapi-manager | `openapi/paths/numbers/id_metadata.yaml` | New: PUT endpoint definition |
| bin-call-manager | `pkg/callhandler/start.go` | Add `num *nmnumber.Number` param + inclusive OR rtp_debug |
| bin-call-manager | `pkg/callhandler/start_incoming_domain_type_sip.go` | Pass `&numb` / `nil` to startCallTypeFlow |
| bin-call-manager | `pkg/callhandler/start_incoming_domain_type_registrar.go` | Pass `nil` to all 4 startCallTypeFlow calls |
| bin-call-manager | `pkg/callhandler/start_incoming_domain_type_trunk.go` | Pass `nil` to startCallTypeFlow |
| bin-api-manager | `docsdev/source/number_struct_number.rst` | Add metadata to struct doc |
| bin-api-manager | `docsdev/source/number_tutorial.rst` | Add Update number metadata tutorial |
| bin-api-manager | `docsdev/build/` | Rebuilt HTML |
