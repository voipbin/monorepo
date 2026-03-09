# Call RTP Debug Capture — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire Customer.Metadata.RTPDebug to trigger RTPEngine start/stop recording commands during call lifecycle, and expose the debug state on the call via a new Metadata field.

**Architecture:** Two trigger points in callhandler (incoming at call creation, outgoing at answer), one stop point at hangup. Call model gets a flexible `map[string]interface{}` metadata field. Channel model gets typed SIPData key constants. Customer model gets MetadataKey constants.

**Tech Stack:** Go, MySQL (Alembic migration), OpenAPI/oapi-codegen, Sphinx RST docs

**Design doc:** `docs/plans/2026-03-09-call-rtp-debug-capture-design.md`

---

### Task 1: Alembic migration — add metadata column to call_calls

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/f3a4b5c6d7e8_call_calls_add_column_metadata.py`

**Step 1: Create the migration file**

```python
"""call_calls_add_column_metadata

Revision ID: f3a4b5c6d7e8
Revises: e2f3a4b5c6d7
Create Date: 2026-03-09 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f3a4b5c6d7e8'
down_revision = 'e2f3a4b5c6d7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE call_calls ADD metadata JSON DEFAULT NULL AFTER data;""")


def downgrade():
    op.execute("""ALTER TABLE call_calls DROP COLUMN metadata;""")
```

Pattern follows `e2f3a4b5c6d7_call_channels_add_column_sip_data.py` (most recent migration).

**Step 2: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/f3a4b5c6d7e8_call_calls_add_column_metadata.py
git commit -m "NOJIRA-Add-call-rtp-debug-capture

- bin-dbscheme-manager: Add metadata JSON column to call_calls table"
```

**IMPORTANT:** Do NOT run `alembic upgrade`. Migration files are committed only — applied by human.

---

### Task 2: Customer model — add MetadataKey constants

**Files:**
- Modify: `bin-customer-manager/models/customer/metadata.go`

**Step 1: Add MetadataKey type and constant**

Current file (lines 1-7):
```go
package customer

// Metadata holds internal-use configuration flags for a customer.
// Managed exclusively by ProjectSuperAdmin. Not exposed in WebhookMessage.
type Metadata struct {
	RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
```

Add above the struct:
```go
// MetadataKey defines typed keys for customer metadata fields.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug enables RTPEngine RTP capture (PCAP) for this customer's calls.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)
```

**Step 2: Run verification**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass (no consumers of the new constant yet).

**Step 3: Commit**

```bash
git add bin-customer-manager/models/customer/metadata.go
git commit -m "NOJIRA-Add-call-rtp-debug-capture

- bin-customer-manager: Add MetadataKey type and MetadataKeyRTPDebug constant"
```

---

### Task 3: Channel model — add SIPData key constants

**Files:**
- Create: `bin-call-manager/models/channel/sipdata.go`

**Step 1: Create the constants file**

```go
package channel

// SIPDataKey defines typed keys for Channel.SIPData map entries.
// Values are populated from Kamailio's Redis hash (kamailio:<sip-call-id>).
type SIPDataKey = string

const (
	SIPDataKeyCallID           SIPDataKey = "call_id"
	SIPDataKeyFromUser         SIPDataKey = "from_user"
	SIPDataKeyFromName         SIPDataKey = "from_name"
	SIPDataKeyFromDomain       SIPDataKey = "from_domain"
	SIPDataKeyFromURI          SIPDataKey = "from_uri"
	SIPDataKeyToUser           SIPDataKey = "to_user"
	SIPDataKeyToName           SIPDataKey = "to_name"
	SIPDataKeyToDomain         SIPDataKey = "to_domain"
	SIPDataKeyToURI            SIPDataKey = "to_uri"
	SIPDataKeyPAI              SIPDataKey = "pai"
	SIPDataKeyRTPEngineAddress SIPDataKey = "rtpengine_address"
	SIPDataKeyDirection        SIPDataKey = "direction"
	SIPDataKeySourceIP         SIPDataKey = "source_ip"
	SIPDataKeyTransport        SIPDataKey = "transport"
	SIPDataKeyDomain           SIPDataKey = "domain"
)
```

**Step 2: Verify** (no test needed — pure constants, covered by build)

```bash
cd bin-call-manager
go build ./...
```

Expected: Build succeeds.

---

### Task 4: Call model — add Metadata field, MetadataKey, Field constant, WebhookMessage

**Files:**
- Modify: `bin-call-manager/models/call/call.go` (add Metadata field)
- Create: `bin-call-manager/models/call/metadata.go` (MetadataKey constants)
- Modify: `bin-call-manager/models/call/field.go` (add FieldMetadata)
- Modify: `bin-call-manager/models/call/webhook.go` (add Metadata to WebhookMessage)

**Step 1: Create `metadata.go`**

```go
package call

// MetadataKey defines typed keys for Call.Metadata map entries.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug indicates RTP debug capture was enabled for this call.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)
```

**Step 2: Add Metadata field to Call struct**

In `call.go`, add after the `Data` field (line 48):
```go
Data           map[DataType]string    `json:"data,omitempty" db:"data,json"`
Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata,json"`
```

**Step 3: Add FieldMetadata to `field.go`**

Add after `FieldData` (line 30):
```go
FieldData     Field = "data"
FieldMetadata Field = "metadata"
```

**Step 4: Add Metadata to WebhookMessage in `webhook.go`**

Add after the `Action` field (around line 38):
```go
Action    fmaction.Action `json:"action,omitempty"`
Metadata  map[string]interface{} `json:"metadata,omitempty"`
```

Update `ConvertWebhookMessage()` to copy the field. Add inside the return struct:
```go
Metadata:  c.Metadata,
```

**Step 5: Verify build**

```bash
cd bin-call-manager
go build ./...
```

---

### Task 5: Call dbhandler — initialize nil Metadata

**Files:**
- Modify: `bin-call-manager/pkg/dbhandler/call.go`

**Step 1: Add nil Metadata initialization in `callGetFromRow()`**

In `callGetFromRow()` (around line 32-46), after existing nil-checks, add:
```go
if res.Metadata == nil {
    res.Metadata = map[string]interface{}{}
}
```

Follow the existing pattern — add after `if res.Data == nil { res.Data = map[call.DataType]string{} }`.

**Step 2: Add nil Metadata initialization in `CallCreate()`**

In `CallCreate()` (around lines 64-75), after existing nil-checks, add:
```go
if c.Metadata == nil {
    c.Metadata = map[string]interface{}{}
}
```

**Step 3: Verify**

```bash
cd bin-call-manager
go build ./...
go test ./pkg/dbhandler/...
```

---

### Task 6: Call handler — add RTP debug helper function

**Files:**
- Create: `bin-call-manager/pkg/callhandler/rtpdebug.go`

**Step 1: Create the helper functions**

This file contains shared logic for starting and stopping RTP debug recording. Keeping it in a separate file avoids cluttering `start.go`, `arievent.go`, and `hangup.go`.

```go
package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// rtpDebugStartRecording sends "start recording" to RTPEngine if the customer has RTP debug enabled.
// Best-effort: logs errors but does not return them (must not block call flow).
func (h *callHandler) rtpDebugStartRecording(ctx context.Context, cn *channel.Channel) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "rtpDebugStartRecording",
		"channel_id": cn.ID,
	})

	rtpengineAddress := cn.SIPData[channel.SIPDataKeyRTPEngineAddress]
	if rtpengineAddress == "" {
		log.Debugf("No rtpengine_address in SIPData. Skipping RTP debug start.")
		return
	}

	command := map[string]interface{}{
		"command": "start recording",
		"call-id": cn.SIPCallID,
	}

	res, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, command)
	if err != nil {
		log.Errorf("Could not send start recording to RTPEngine. rtpengine_address: %s, err: %v", rtpengineAddress, err)
		return
	}
	log.WithField("response", res).Debugf("Sent start recording to RTPEngine. rtpengine_address: %s, sip_call_id: %s", rtpengineAddress, cn.SIPCallID)
}

// rtpDebugStopRecording sends "stop recording" to RTPEngine for a call that had RTP debug enabled.
// Fetches a fresh channel from DB (hangup channel may be stale).
// Best-effort: logs errors but does not return them (must not block hangup flow).
func (h *callHandler) rtpDebugStopRecording(ctx context.Context, c *call.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "rtpDebugStopRecording",
		"call_id": c.ID,
	})

	cn, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get fresh channel for RTP debug stop. channel_id: %s, err: %v", c.ChannelID, err)
		return
	}
	log.WithField("channel", cn).Debugf("Retrieved fresh channel for RTP debug stop. channel_id: %s", cn.ID)

	rtpengineAddress := cn.SIPData[channel.SIPDataKeyRTPEngineAddress]
	if rtpengineAddress == "" {
		log.Debugf("No rtpengine_address in SIPData. Skipping RTP debug stop.")
		return
	}

	command := map[string]interface{}{
		"command": "stop recording",
		"call-id": cn.SIPCallID,
	}

	res, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, command)
	if err != nil {
		log.Errorf("Could not send stop recording to RTPEngine. rtpengine_address: %s, err: %v", rtpengineAddress, err)
		return
	}
	log.WithField("response", res).Debugf("Sent stop recording to RTPEngine. rtpengine_address: %s, sip_call_id: %s", rtpengineAddress, cn.SIPCallID)
}
```

**Step 2: Verify build**

```bash
cd bin-call-manager
go build ./...
```

---

### Task 7: Call handler — incoming call RTP debug trigger

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start.go` (in `startCallTypeFlow()`)

**Step 1: Add RTP debug check after call creation**

In `startCallTypeFlow()`, after the call is created (line 643: `log.WithField("call", c).Debugf("Created a call...")`), add the RTP debug check before `setVariablesCall`:

```go
log.WithField("call", c).Debugf("Created a call. call: %s", c.ID)

// RTP debug: check if customer has RTP debug enabled
cs, errCS := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
if errCS != nil {
    log.Errorf("Could not get customer for RTP debug check. customer_id: %s, err: %v", customerID, errCS)
} else {
    log.WithField("customer", cs).Debugf("Retrieved customer for RTP debug check. customer_id: %s", cs.ID)
    if cs.Metadata.RTPDebug {
        c.Metadata[call.MetadataKeyRTPDebug] = true
        if errMeta := h.db.CallUpdate(ctx, c.ID, map[call.Field]any{
            call.FieldMetadata: c.Metadata,
        }); errMeta != nil {
            log.Errorf("Could not update call metadata for RTP debug. err: %v", errMeta)
        } else {
            h.rtpDebugStartRecording(ctx, cn)
        }
    }
}

// set variables
```

**Step 2: Verify build**

```bash
cd bin-call-manager
go build ./...
```

---

### Task 8: Call handler — outgoing call RTP debug trigger

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/status.go` (in `updateStatusProgressing()`)

**Step 1: Add RTP debug check in updateStatusProgressing**

In `updateStatusProgressing()` (around line 43-91), after the status update succeeds and the call is fetched but before ActionNext, add the RTP debug check for outgoing calls only. The outgoing call direction check: incoming calls return early at line 64, so code after that line only runs for outgoing calls.

After the early return for incoming calls (line 66), add the RTP debug check:

```go
if res.Direction == call.DirectionIncoming {
    return nil
}

// RTP debug: check if customer has RTP debug enabled (outgoing calls)
cs, errCS := h.reqHandler.CustomerV1CustomerGet(ctx, res.CustomerID)
if errCS != nil {
    log.Errorf("Could not get customer for RTP debug check. customer_id: %s, err: %v", res.CustomerID, errCS)
} else {
    log.WithField("customer", cs).Debugf("Retrieved customer for RTP debug check. customer_id: %s", cs.ID)
    if cs.Metadata.RTPDebug {
        res.Metadata[call.MetadataKeyRTPDebug] = true
        if errMeta := h.db.CallUpdate(ctx, res.ID, map[call.Field]any{
            call.FieldMetadata: res.Metadata,
        }); errMeta != nil {
            log.Errorf("Could not update call metadata for RTP debug. err: %v", errMeta)
        } else {
            h.rtpDebugStartRecording(ctx, cn)
        }
    }
}
```

Note: `cn` is the channel parameter passed to `updateStatusProgressing()`. Verify the function signature has access to the channel — if not, it's available from calling `h.channelHandler.Get(ctx, res.ChannelID)`.

**Step 2: Verify build**

```bash
cd bin-call-manager
go build ./...
```

---

### Task 9: Call handler — hangup RTP debug stop

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/hangup.go` (in `Hangup()`)

**Step 1: Add RTP debug stop check**

In `Hangup()`, after `UpdateHangupInfo` returns (line 62, after `res, err := h.UpdateHangupInfo(ctx, c.ID, reason, hangupBy)`), add:

```go
res, err := h.UpdateHangupInfo(ctx, c.ID, reason, hangupBy)
if err != nil {
    log.Errorf("Could not update hangup info. err: %v", err)
    return nil, err
}

// RTP debug: stop recording if enabled for this call
if v, ok := res.Metadata[call.MetadataKeyRTPDebug]; ok && v == true {
    h.rtpDebugStopRecording(ctx, res)
}
```

**Step 2: Verify build**

```bash
cd bin-call-manager
go build ./...
```

---

### Task 10: Run full verification for bin-call-manager

**Step 1: Run complete verification**

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass. Fix any issues before proceeding.

**Step 2: Commit all bin-call-manager changes**

```bash
git add bin-call-manager/models/call/metadata.go
git add bin-call-manager/models/call/call.go
git add bin-call-manager/models/call/field.go
git add bin-call-manager/models/call/webhook.go
git add bin-call-manager/models/channel/sipdata.go
git add bin-call-manager/pkg/dbhandler/call.go
git add bin-call-manager/pkg/callhandler/rtpdebug.go
git add bin-call-manager/pkg/callhandler/start.go
git add bin-call-manager/pkg/callhandler/status.go
git add bin-call-manager/pkg/callhandler/hangup.go
git commit -m "NOJIRA-Add-call-rtp-debug-capture

- bin-call-manager: Add Metadata field to Call model with MetadataKey constants
- bin-call-manager: Add SIPData key constants for Channel model
- bin-call-manager: Add RTP debug start recording for incoming calls at creation
- bin-call-manager: Add RTP debug start recording for outgoing calls at answer
- bin-call-manager: Add RTP debug stop recording at hangup
- bin-call-manager: Initialize nil Metadata as empty map in dbhandler"
```

---

### Task 11: OpenAPI schema — add metadata field to CallManagerCall

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (CallManagerCall schema, around line 920)

**Step 1: Add metadata property**

After the `hangup_reason` field (around line 884), add:

```yaml
        metadata:
          type: object
          additionalProperties: true
          nullable: true
          description: |
            Internal metadata for the call. Contains key-value pairs set by the system.
            Currently supported keys:
            - `rtp_debug` (boolean): When `true`, RTPEngine is capturing RTP traffic for this call.
          example:
            rtp_debug: true
```

**Step 2: Regenerate models**

```bash
cd bin-openapi-manager
go generate ./...
```

**Step 3: Run verification**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Regenerate api-manager server code**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

**Step 5: Run api-manager verification**

```bash
cd bin-api-manager
go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git add bin-openapi-manager/gens/models/gen.go
git add bin-api-manager/gens/openapi_server/gen.go
git commit -m "NOJIRA-Add-call-rtp-debug-capture

- bin-openapi-manager: Add metadata field to CallManagerCall schema
- bin-api-manager: Regenerate server code with metadata field"
```

---

### Task 12: RST documentation — add metadata field to call struct docs

**Files:**
- Modify: `bin-api-manager/docsdev/source/call_struct_call.rst`

**Step 1: Add metadata to the struct example**

In the JSON example block, add `"metadata"` after `"hangup_reason"`:

```rst
        "hangup_reason": "<string>",
        "metadata": {
            ...
        },
        "tm_create": "<string>",
```

**Step 2: Add field description**

After the `hangup_reason` field description, add:

```rst
* ``metadata`` (Object, nullable): Internal metadata for the call. Contains key-value pairs set by the system. Currently supported keys: ``rtp_debug`` (boolean) — when ``true``, RTPEngine is capturing RTP traffic for this call. May be empty ``{}`` if no metadata is set.
```

**Step 3: Rebuild HTML**

```bash
cd bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

**Step 4: Commit**

```bash
git add bin-api-manager/docsdev/source/call_struct_call.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Add-call-rtp-debug-capture

- bin-api-manager: Add metadata field to call struct RST documentation
- bin-api-manager: Rebuild HTML docs"
```

---

### Task 13: Final verification and conflict check

**Step 1: Run verification for all changed services**

```bash
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Add-call-rtp-debug-capture
```

Create PR with title: `NOJIRA-Add-call-rtp-debug-capture`
