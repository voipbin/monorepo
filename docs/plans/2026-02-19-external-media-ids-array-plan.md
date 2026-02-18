# ExternalMediaID → ExternalMediaIDs Array Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace singular `ExternalMediaID uuid.UUID` with `ExternalMediaIDs []uuid.UUID` array in both Call and Confbridge models, supporting multiple concurrent external media streams per resource.

**Architecture:** JSON column pattern (matching `ChainedCallIDs`, `RecordingIDs`) with MySQL `json_array_append`/`json_remove`+`json_search` for atomic array operations. Max 5 external medias per call/confbridge. ExternalMediaStop via API targets a specific ID; flow action `external_media_stop` stops ALL.

**Tech Stack:** Go, MySQL JSON functions, Squirrel query builder, gomock

---

### Task 1: Update Call Model and Field Constant

**Files:**
- Modify: `bin-call-manager/models/call/call.go:39`
- Modify: `bin-call-manager/models/call/field.go:23`

**Step 1: Update the Call struct field**

In `bin-call-manager/models/call/call.go`, replace line 39:

```go
// Before:
ExternalMediaID uuid.UUID `json:"external_media_id,omitempty" db:"external_media_id,uuid"` // external media id(current)

// After:
ExternalMediaIDs []uuid.UUID `json:"external_media_ids,omitempty" db:"external_media_ids,json"` // external media ids
```

**Step 2: Update the field constant**

In `bin-call-manager/models/call/field.go`, replace line 23:

```go
// Before:
FieldExternalMediaID Field = "external_media_id" // external_media_id

// After:
FieldExternalMediaIDs Field = "external_media_ids" // external_media_ids
```

**Step 3: Commit**

```bash
git add bin-call-manager/models/call/call.go bin-call-manager/models/call/field.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Change Call.ExternalMediaID to ExternalMediaIDs []uuid.UUID array
- bin-call-manager: Update field constant to FieldExternalMediaIDs"
```

---

### Task 2: Update Confbridge Model and Field Constant

**Files:**
- Modify: `bin-call-manager/models/confbridge/main.go:33`
- Modify: `bin-call-manager/models/confbridge/field.go:24`

**Step 1: Update the Confbridge struct field**

In `bin-call-manager/models/confbridge/main.go`, replace line 33:

```go
// Before:
ExternalMediaID uuid.UUID `json:"external_media_id" db:"external_media_id,uuid"`

// After:
ExternalMediaIDs []uuid.UUID `json:"external_media_ids" db:"external_media_ids,json"`
```

**Step 2: Update the field constant**

In `bin-call-manager/models/confbridge/field.go`, replace line 24:

```go
// Before:
FieldExternalMediaID Field = "external_media_id" // external_media_id

// After:
FieldExternalMediaIDs Field = "external_media_ids" // external_media_ids
```

**Step 3: Commit**

```bash
git add bin-call-manager/models/confbridge/main.go bin-call-manager/models/confbridge/field.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Change Confbridge.ExternalMediaID to ExternalMediaIDs []uuid.UUID array
- bin-call-manager: Update confbridge field constant to FieldExternalMediaIDs"
```

---

### Task 3: Update Test SQL Scripts

**Files:**
- Modify: `bin-call-manager/scripts/database_scripts_test/table_calls.sql:24,70`
- Modify: `bin-call-manager/scripts/database_scripts_test/table_confbridges.sql:22`

**Step 1: Update call_calls test schema**

In `table_calls.sql`, replace:
```sql
-- Before:
  external_media_id binary(16), -- external media id

-- After:
  external_media_ids json, -- external media ids
```

Remove the index line:
```sql
-- Remove this line:
create index idx_call_calls_external_media_id on call_calls(external_media_id);
```

**Step 2: Update call_confbridges test schema**

In `table_confbridges.sql`, replace:
```sql
-- Before:
  external_media_id binary(16),

-- After:
  external_media_ids json,
```

**Step 3: Commit**

```bash
git add bin-call-manager/scripts/database_scripts_test/table_calls.sql bin-call-manager/scripts/database_scripts_test/table_confbridges.sql
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Update test SQL schemas for external_media_ids JSON column"
```

---

### Task 4: Update DB Handler — Call Operations

**Files:**
- Modify: `bin-call-manager/pkg/dbhandler/main.go:56`
- Modify: `bin-call-manager/pkg/dbhandler/call.go:24-46,374-450`

**Step 1: Update DBHandler interface**

In `pkg/dbhandler/main.go`, replace:
```go
// Before:
CallSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error

// After:
CallAddExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error
CallRemoveExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error
```

**Step 2: Add nil slice init in callGetFromRow**

In `pkg/dbhandler/call.go`, after the `res.RecordingIDs == nil` block (after line 37), add:
```go
if res.ExternalMediaIDs == nil {
    res.ExternalMediaIDs = []uuid.UUID{}
}
```

**Step 3: Add nil slice init in CallCreate**

In `pkg/dbhandler/call.go`, after the `c.RecordingIDs == nil` block (after line 65), add:
```go
if c.ExternalMediaIDs == nil {
    c.ExternalMediaIDs = []uuid.UUID{}
}
```

**Step 4: Replace CallSetExternalMediaID with CallAddExternalMediaID and CallRemoveExternalMediaID**

In `pkg/dbhandler/call.go`, replace the `CallSetExternalMediaID` function (lines 445-450) with:

```go
// CallAddExternalMediaID adds the external media id to the call's external_media_ids.
func (h *handler) CallAddExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error {
	q := `
	update call_calls set
		external_media_ids = json_array_append(
			external_media_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.String(), h.utilHandler.TimeNow(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddExternalMediaID. err: %v", err)
	}

	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallRemoveExternalMediaID removes the external media id from the call's external_media_ids.
func (h *handler) CallRemoveExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error {
	q := `
	update call_calls set
		external_media_ids = json_remove(
			external_media_ids, replace(
				json_search(
					external_media_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.String(), h.utilHandler.TimeNow(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveExternalMediaID. err: %v", err)
	}

	_ = h.callUpdateToCache(ctx, id)

	return nil
}
```

**Step 5: Run tests**

Run: `cd bin-call-manager && go test ./pkg/dbhandler/... -v -count=1`
Expected: Tests should compile (some may fail due to mock mismatch — will be fixed after mock regen)

**Step 6: Commit**

```bash
git add bin-call-manager/pkg/dbhandler/main.go bin-call-manager/pkg/dbhandler/call.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Replace CallSetExternalMediaID with CallAddExternalMediaID/CallRemoveExternalMediaID
- bin-call-manager: Add ExternalMediaIDs nil slice init in callGetFromRow and CallCreate"
```

---

### Task 5: Update DB Handler — Confbridge Operations

**Files:**
- Modify: `bin-call-manager/pkg/dbhandler/main.go:109`
- Modify: `bin-call-manager/pkg/dbhandler/confbridge.go:24-63,340-345`

**Step 1: Update DBHandler interface**

In `pkg/dbhandler/main.go`, replace:
```go
// Before:
ConfbridgeSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error

// After:
ConfbridgeAddExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error
ConfbridgeRemoveExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error
```

**Step 2: Add nil slice init in confbridgeGetFromRow**

In `pkg/dbhandler/confbridge.go`, after the `res.RecordingIDs == nil` block (after line 37), add:
```go
if res.ExternalMediaIDs == nil {
    res.ExternalMediaIDs = []uuid.UUID{}
}
```

**Step 3: Add nil slice init in ConfbridgeCreate**

In `pkg/dbhandler/confbridge.go`, after the `cb.RecordingIDs == nil` block (after line 63), add:
```go
if cb.ExternalMediaIDs == nil {
    cb.ExternalMediaIDs = []uuid.UUID{}
}
```

**Step 4: Replace ConfbridgeSetExternalMediaID with add/remove functions**

In `pkg/dbhandler/confbridge.go`, replace `ConfbridgeSetExternalMediaID` (lines 340-345) with:

```go
// ConfbridgeAddExternalMediaID adds the external media id to the confbridge's external_media_ids.
func (h *handler) ConfbridgeAddExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error {
	q := `
	update call_confbridges set
		external_media_ids = json_array_append(
			external_media_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.String(), h.utilHandler.TimeNow(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeAddExternalMediaID. err: %v", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}

// ConfbridgeRemoveExternalMediaID removes the external media id from the confbridge's external_media_ids.
func (h *handler) ConfbridgeRemoveExternalMediaID(ctx context.Context, id, externalMediaID uuid.UUID) error {
	q := `
	update call_confbridges set
		external_media_ids = json_remove(
			external_media_ids, replace(
				json_search(
					external_media_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.String(), h.utilHandler.TimeNow(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeRemoveExternalMediaID. err: %v", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}
```

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/dbhandler/main.go bin-call-manager/pkg/dbhandler/confbridge.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Replace ConfbridgeSetExternalMediaID with add/remove operations
- bin-call-manager: Add ExternalMediaIDs nil slice init in confbridgeGetFromRow and ConfbridgeCreate"
```

---

### Task 6: Update Call Handler — Replace UpdateExternalMediaID, Update ExternalMediaStart/Stop

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/main.go:59,132`
- Modify: `bin-call-manager/pkg/callhandler/db.go:82-88,331-352`
- Modify: `bin-call-manager/pkg/callhandler/external_media.go`

**Step 1: Update CallHandler interface**

In `pkg/callhandler/main.go`:

Remove `UpdateExternalMediaID` from the interface (line 59):
```go
// Remove this line:
UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error)
```

Update `ExternalMediaStop` signature (line 132):
```go
// Before:
ExternalMediaStop(ctx context.Context, id uuid.UUID) (*call.Call, error)

// After:
ExternalMediaStop(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error)
```

**Step 2: Replace UpdateExternalMediaID in db.go**

In `pkg/callhandler/db.go`, replace the `UpdateExternalMediaID` function (lines 331-352) with:

```go
// AddExternalMediaID adds an external media ID to the call's external_media_ids array.
func (h *callHandler) AddExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "AddExternalMediaID",
		"call_id":           id,
		"external_media_id": externalMediaID,
	})

	if errAdd := h.db.CallAddExternalMediaID(ctx, id, externalMediaID); errAdd != nil {
		log.Errorf("Could not add the external media id. err: %v", errAdd)
		return nil, errAdd
	}

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RemoveExternalMediaID removes an external media ID from the call's external_media_ids array.
func (h *callHandler) RemoveExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "RemoveExternalMediaID",
		"call_id":           id,
		"external_media_id": externalMediaID,
	})

	if errRemove := h.db.CallRemoveExternalMediaID(ctx, id, externalMediaID); errRemove != nil {
		log.Errorf("Could not remove the external media id. err: %v", errRemove)
		return nil, errRemove
	}

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 3: Update ExternalMediaStart in external_media.go**

Add the max limit constant at the top of the file:
```go
const defaultMaxExternalMediaPerCall = 5
```

Replace the guard check and update call:
```go
// Before:
if c.ExternalMediaID != uuid.Nil {
    log.Errorf("The call has external media already. external_media_id: %s", c.ExternalMediaID)
    return nil, fmt.Errorf("the call has external media already")
}

// After:
if len(c.ExternalMediaIDs) >= defaultMaxExternalMediaPerCall {
    log.Errorf("The call has reached the maximum number of external medias. count: %d", len(c.ExternalMediaIDs))
    return nil, fmt.Errorf("the call has reached the maximum number of external medias")
}
```

Replace the update call:
```go
// Before:
res, err := h.UpdateExternalMediaID(ctx, id, tmp.ID)

// After:
res, err := h.AddExternalMediaID(ctx, id, tmp.ID)
```

**Step 4: Update ExternalMediaStop in external_media.go**

Replace the entire function:
```go
// ExternalMediaStop stops a specific external media on the call
func (h *callHandler) ExternalMediaStop(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExternalMediaStop",
		"call_id":           id,
		"external_media_id": externalMediaID,
	})
	log.Debug("Stopping the external media.")

	// get call
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	// validate the external media ID exists in the call's list
	found := false
	for _, emID := range c.ExternalMediaIDs {
		if emID == externalMediaID {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("The external media id is not in the call's external media ids. external_media_id: %s", externalMediaID)
		return nil, fmt.Errorf("the external media id is not associated with this call")
	}

	tmp, err := h.externalMediaHandler.Stop(ctx, externalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media handler. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)

	// remove from array
	res, err := h.RemoveExternalMediaID(ctx, id, externalMediaID)
	if err != nil {
		log.Errorf("Could not remove the external media id. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 5: Update call Create in db.go**

In `pkg/callhandler/db.go`, in the `Create` function, replace:
```go
// Before:
ExternalMediaID: uuid.Nil,

// After:
ExternalMediaIDs: []uuid.UUID{},
```

**Step 6: Commit**

```bash
git add bin-call-manager/pkg/callhandler/main.go bin-call-manager/pkg/callhandler/db.go bin-call-manager/pkg/callhandler/external_media.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Replace UpdateExternalMediaID with AddExternalMediaID/RemoveExternalMediaID
- bin-call-manager: Update ExternalMediaStart to check max limit instead of single ID
- bin-call-manager: Update ExternalMediaStop to accept specific externalMediaID parameter"
```

---

### Task 7: Update Confbridge Handler

**Files:**
- Modify: `bin-call-manager/pkg/confbridgehandler/main.go:50,83`
- Modify: `bin-call-manager/pkg/confbridgehandler/db.go:133-147`
- Modify: `bin-call-manager/pkg/confbridgehandler/external_media.go`

**Step 1: Update ConfbridgeHandler interface**

In `pkg/confbridgehandler/main.go`:

Replace `UpdateExternalMediaID` (line 50):
```go
// Remove this line:
UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error)
```

Update `ExternalMediaStop` signature (line 83):
```go
// Before:
ExternalMediaStop(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)

// After:
ExternalMediaStop(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error)
```

**Step 2: Replace UpdateExternalMediaID in db.go with Add/Remove**

In `pkg/confbridgehandler/db.go`, replace `UpdateExternalMediaID` (lines 133-147) with:

```go
// AddExternalMediaID adds an external media ID to the confbridge's external_media_ids array.
func (h *confbridgeHandler) AddExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {
	if errAdd := h.db.ConfbridgeAddExternalMediaID(ctx, id, externalMediaID); errAdd != nil {
		return nil, errors.Wrapf(errAdd, "could not add the external media id. confbridge_id: %s, external_media_id: %s", id, externalMediaID)
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the updated confbridge. confbridge_id: %s", id)
	}

	return res, nil
}

// RemoveExternalMediaID removes an external media ID from the confbridge's external_media_ids array.
func (h *confbridgeHandler) RemoveExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {
	if errRemove := h.db.ConfbridgeRemoveExternalMediaID(ctx, id, externalMediaID); errRemove != nil {
		return nil, errors.Wrapf(errRemove, "could not remove the external media id. confbridge_id: %s, external_media_id: %s", id, externalMediaID)
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the updated confbridge. confbridge_id: %s", id)
	}

	return res, nil
}
```

**Step 3: Update ExternalMediaStart in external_media.go**

Add the max limit constant:
```go
const defaultMaxExternalMediaPerConfbridge = 5
```

Replace the guard check:
```go
// Before:
if c.ExternalMediaID != uuid.Nil {
    log.Errorf("The confbridge has external media already. external_media_id: %s", c.ExternalMediaID)
    return nil, fmt.Errorf("the confbridge has external media already")
}

// After:
if len(c.ExternalMediaIDs) >= defaultMaxExternalMediaPerConfbridge {
    log.Errorf("The confbridge has reached the maximum number of external medias. count: %d", len(c.ExternalMediaIDs))
    return nil, fmt.Errorf("the confbridge has reached the maximum number of external medias")
}
```

Replace the update call:
```go
// Before:
res, err := h.UpdateExternalMediaID(ctx, id, tmp.ID)

// After:
res, err := h.AddExternalMediaID(ctx, id, tmp.ID)
```

**Step 4: Update ExternalMediaStop in external_media.go**

Replace the entire function:
```go
// ExternalMediaStop stops a specific external media on the confbridge
func (h *confbridgeHandler) ExternalMediaStop(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExternalMediaStop",
		"confbridge_id":     id,
		"external_media_id": externalMediaID,
	})
	log.Debug("Stopping the external media.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, err
	}

	found := false
	for _, emID := range c.ExternalMediaIDs {
		if emID == externalMediaID {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("The external media id is not in the confbridge's external media ids. external_media_id: %s", externalMediaID)
		return nil, fmt.Errorf("the external media id is not associated with this confbridge")
	}

	tmp, err := h.externalMediaHandler.Stop(ctx, externalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media handler. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)

	res, err := h.RemoveExternalMediaID(ctx, id, externalMediaID)
	if err != nil {
		log.Errorf("Could not remove the external media id. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/confbridgehandler/main.go bin-call-manager/pkg/confbridgehandler/db.go bin-call-manager/pkg/confbridgehandler/external_media.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Replace confbridge UpdateExternalMediaID with Add/RemoveExternalMediaID
- bin-call-manager: Update confbridge ExternalMediaStart/Stop for array pattern"
```

---

### Task 8: Update External Media Handler — Stop Removes from Parent

**Files:**
- Modify: `bin-call-manager/pkg/externalmediahandler/stop.go`

**Step 1: Update Stop() to remove from parent array**

After the external media delete, add parent cleanup:

```go
func (h *externalMediaHandler) Stop(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Stop",
		"external_media_id": externalMediaID,
	})
	log.Debug("Stopping the external media.")

	res, err := h.UpdateStatus(ctx, externalMediaID, externalmedia.StatusTerminating)
	if err != nil {
		return nil, fmt.Errorf("could not update external media status: %w", err)
	}
	promExternalMediaStopTotal.WithLabelValues(string(res.ReferenceType)).Inc()

	// hangup the external media channel
	if errHangup := h.channelHandler.HangingUpWithAsteriskID(ctx, res.AsteriskID, res.ChannelID, ari.ChannelCauseNormalClearing); errHangup != nil {
		return nil, errors.Wrapf(errHangup, "could not hangup the external media channel")
	}

	// delete external media info
	if errExtDelete := h.db.ExternalMediaDelete(ctx, externalMediaID); errExtDelete != nil {
		return nil, errors.Wrapf(errExtDelete, "could not delete external media info from db")
	}

	// remove the external media ID from the parent's ExternalMediaIDs array
	switch res.ReferenceType {
	case externalmedia.ReferenceTypeCall:
		if errRemove := h.db.CallRemoveExternalMediaID(ctx, res.ReferenceID, externalMediaID); errRemove != nil {
			log.Errorf("Could not remove external media id from call. call_id: %s, err: %v", res.ReferenceID, errRemove)
		}
	case externalmedia.ReferenceTypeConfbridge:
		if errRemove := h.db.ConfbridgeRemoveExternalMediaID(ctx, res.ReferenceID, externalMediaID); errRemove != nil {
			log.Errorf("Could not remove external media id from confbridge. confbridge_id: %s, err: %v", res.ReferenceID, errRemove)
		}
	}

	return res, nil
}
```

**Step 2: Commit**

```bash
git add bin-call-manager/pkg/externalmediahandler/stop.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: ExternalMediaHandler.Stop() now removes ID from parent call/confbridge array"
```

---

### Task 9: Update Action Handler

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/action.go:662-724`

**Step 1: Update actionExecuteExternalMediaStart**

In `pkg/callhandler/action.go`, replace the log line (around line 694):
```go
// Before:
log.WithField("call", cc).Debugf("Started external media. external_media_id: %s", cc.ExternalMediaID)

// After:
if len(cc.ExternalMediaIDs) > 0 {
    log.WithField("call", cc).Debugf("Started external media. external_media_id: %s", cc.ExternalMediaIDs[len(cc.ExternalMediaIDs)-1])
} else {
    log.WithField("call", cc).Debugf("Started external media.")
}
```

**Step 2: Update actionExecuteExternalMediaStop**

Replace the function body (lines 701-724):
```go
func (h *callHandler) actionExecuteExternalMediaStop(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteExternalMediaStop",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	if len(c.ExternalMediaIDs) == 0 {
		// nothing to do here
		log.Infof("The call has no external media. call_id: %s", c.ID)
	} else {
		// stop ALL external medias on the call (flow-driven cleanup)
		for _, emID := range c.ExternalMediaIDs {
			tmp, err := h.externalMediaHandler.Stop(ctx, emID)
			if err != nil {
				log.Errorf("Could not stop the external media. external_media_id: %s, err: %v", emID, err)
				// continue stopping others even if one fails
				continue
			}
			log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)
		}
	}

	// send next action request
	return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
}
```

**Step 3: Commit**

```bash
git add bin-call-manager/pkg/callhandler/action.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Update action handlers for ExternalMediaIDs array
- bin-call-manager: Flow action external_media_stop now stops ALL external medias"
```

---

### Task 10: Update Listen Handler — Parse ExternalMediaID from Delete Body

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/models/request/calls.go`
- Modify: `bin-call-manager/pkg/listenhandler/v1_calls.go:515-547`
- Modify: `bin-call-manager/pkg/listenhandler/v1_confbridges.go:255-285`

**Step 1: Add delete request struct**

In `pkg/listenhandler/models/request/calls.go`, add after the Post struct:
```go
// V1DataCallsIDExternalMediaDelete is
// v1 data type for
// /v1/calls/<call-id>/external-media DELETE
type V1DataCallsIDExternalMediaDelete struct {
	ExternalMediaID uuid.UUID `json:"external_media_id,omitempty"`
}
```

**Step 2: Update processV1CallsIDExternalMediaDelete**

In `pkg/listenhandler/v1_calls.go`, replace the function:
```go
func (h *listenHandler) processV1CallsIDExternalMediaDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDExternalMediaDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDExternalMediaDelete
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the request body. err: %v", err)
		return simpleResponse(400), nil
	}

	if req.ExternalMediaID == uuid.Nil {
		log.Errorf("The external_media_id is required.")
		return simpleResponse(400), nil
	}

	tmp, err := h.callHandler.ExternalMediaStop(ctx, id, req.ExternalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media. call_id: %s, err: %v", id, err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

**Step 3: Update processV1ConfbridgesIDExternalMediaDelete**

Same pattern — parse `external_media_id` from body and pass to `ExternalMediaStop`.

In `pkg/listenhandler/v1_confbridges.go`, update the delete handler similarly:
```go
func (h *listenHandler) processV1ConfbridgesIDExternalMediaDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDExternalMediaDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDExternalMediaDelete
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the request body. err: %v", err)
		return simpleResponse(400), nil
	}

	if req.ExternalMediaID == uuid.Nil {
		log.Errorf("The external_media_id is required.")
		return simpleResponse(400), nil
	}

	tmp, err := h.confbridgeHandler.ExternalMediaStop(ctx, id, req.ExternalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media. confbridge_id: %s, err: %v", id, err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

**Step 4: Commit**

```bash
git add bin-call-manager/pkg/listenhandler/models/request/calls.go bin-call-manager/pkg/listenhandler/v1_calls.go bin-call-manager/pkg/listenhandler/v1_confbridges.go
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Listen handlers now parse external_media_id from DELETE request body
- bin-call-manager: Add V1DataCallsIDExternalMediaDelete request struct"
```

---

### Task 11: Update Request Handler (bin-common-handler)

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:420,517`
- Modify: `bin-common-handler/pkg/requesthandler/call_calls.go:364-381`
- Modify: `bin-common-handler/pkg/requesthandler/call_confbridge.go:161-175`

**Step 1: Update RequestHandler interface**

In `pkg/requesthandler/main.go`, update both signatures:
```go
// Before:
CallV1CallExternalMediaStop(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)

// After:
CallV1CallExternalMediaStop(ctx context.Context, callID uuid.UUID, externalMediaID uuid.UUID) (*cmcall.Call, error)
```

```go
// Before:
CallV1ConfbridgeExternalMediaStop(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error)

// After:
CallV1ConfbridgeExternalMediaStop(ctx context.Context, confbridgeID uuid.UUID, externalMediaID uuid.UUID) (*cmconfbridge.Confbridge, error)
```

**Step 2: Update CallV1CallExternalMediaStop implementation**

In `pkg/requesthandler/call_calls.go`, update the function:
```go
func (r *requestHandler) CallV1CallExternalMediaStop(ctx context.Context, callID uuid.UUID, externalMediaID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/external-media", callID)

	data := struct {
		ExternalMediaID uuid.UUID `json:"external_media_id"`
	}{
		ExternalMediaID: externalMediaID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/calls/<call-id>/external-media", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcall.Call
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 3: Update CallV1ConfbridgeExternalMediaStop implementation**

In `pkg/requesthandler/call_confbridge.go`, same pattern:
```go
func (r *requestHandler) CallV1ConfbridgeExternalMediaStop(ctx context.Context, confbridgeID uuid.UUID, externalMediaID uuid.UUID) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/external-media", confbridgeID)

	data := struct {
		ExternalMediaID uuid.UUID `json:"external_media_id"`
	}{
		ExternalMediaID: externalMediaID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/confbridges/<confbridge-id>/external-media", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmconfbridge.Confbridge
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 4: Update all callers of these functions across the monorepo**

Search for all callers and update them to pass the `externalMediaID` parameter. The known callers that go through call/confbridge stop (not direct external media stop) need updating.

Use: `grep -r "CallV1CallExternalMediaStop\|CallV1ConfbridgeExternalMediaStop" --include="*.go" -l` (excluding test and mock files) to find all callers.

**Step 5: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/main.go bin-common-handler/pkg/requesthandler/call_calls.go bin-common-handler/pkg/requesthandler/call_confbridge.go
git commit -m "NOJIRA-external-media-ids-array

- bin-common-handler: Add externalMediaID parameter to CallV1CallExternalMediaStop
- bin-common-handler: Add externalMediaID parameter to CallV1ConfbridgeExternalMediaStop
- bin-common-handler: Send external_media_id in request body as JSON"
```

---

### Task 12: Regenerate Mocks and Fix Compilation

**Step 1: Regenerate mocks for bin-common-handler**

```bash
cd bin-common-handler && go generate ./...
```

**Step 2: Regenerate mocks for bin-call-manager**

```bash
cd bin-call-manager && go generate ./...
```

**Step 3: Fix any remaining compile errors**

Search for any remaining references to `ExternalMediaID` (singular) in bin-call-manager:
```bash
grep -r "ExternalMediaID\b" --include="*.go" bin-call-manager/ | grep -v "mock_" | grep -v "_test.go" | grep -v "vendor/"
```

Fix all remaining references. Also search for `FieldExternalMediaID` and `CallSetExternalMediaID`/`ConfbridgeSetExternalMediaID` in case any were missed.

**Step 4: Fix test files**

Update test files that reference the old field names and function signatures. The key test files:
- `bin-call-manager/pkg/callhandler/external_media_test.go`
- `bin-call-manager/pkg/callhandler/db_test.go`
- `bin-call-manager/pkg/confbridgehandler/external_media_test.go`
- `bin-call-manager/pkg/confbridgehandler/db_test.go`
- `bin-call-manager/pkg/dbhandler/call_test.go`
- `bin-call-manager/pkg/dbhandler/confbridge_test.go`
- `bin-call-manager/pkg/listenhandler/v1_calls_test.go`
- `bin-call-manager/pkg/listenhandler/v1_confbridge_test.go`
- `bin-common-handler/pkg/requesthandler/call_calls_test.go`
- `bin-common-handler/pkg/requesthandler/call_confbridge_test.go`

**Step 5: Commit**

```bash
git add -A
git commit -m "NOJIRA-external-media-ids-array

- bin-call-manager: Regenerate mocks and fix all remaining ExternalMediaID references
- bin-common-handler: Regenerate mocks for updated RequestHandler interface"
```

---

### Task 13: Create Alembic Migration

**Files:**
- Create: `bin-dbscheme-manager/alembic/versions/<new_migration>.py`

**Step 1: Create the migration**

```bash
cd bin-dbscheme-manager
alembic -c alembic.ini revision -m "change external_media_id to external_media_ids json in call_calls and call_confbridges"
```

**Step 2: Write the migration**

```python
def upgrade() -> None:
    # call_calls
    op.drop_index('idx_call_calls_external_media_id', table_name='call_calls')
    op.drop_column('call_calls', 'external_media_id')
    op.add_column('call_calls', sa.Column('external_media_ids', sa.JSON(), nullable=True, server_default='[]'))

    # call_confbridges
    op.drop_column('call_confbridges', 'external_media_id')
    op.add_column('call_confbridges', sa.Column('external_media_ids', sa.JSON(), nullable=True, server_default='[]'))


def downgrade() -> None:
    # call_calls
    op.drop_column('call_calls', 'external_media_ids')
    op.add_column('call_calls', sa.Column('external_media_id', sa.BINARY(16), nullable=True))
    op.create_index('idx_call_calls_external_media_id', 'call_calls', ['external_media_id'])

    # call_confbridges
    op.drop_column('call_confbridges', 'external_media_ids')
    op.add_column('call_confbridges', sa.Column('external_media_id', sa.BINARY(16), nullable=True))
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/alembic/versions/
git commit -m "NOJIRA-external-media-ids-array

- bin-dbscheme-manager: Add migration to change external_media_id to external_media_ids JSON column"
```

---

### Task 14: Update OpenAPI Schemas

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Update Call schema**

Change the `external_media_id` field in the Call schema to `external_media_ids`:
```yaml
# Before:
external_media_id:
  type: string
  format: uuid

# After:
external_media_ids:
  type: array
  items:
    type: string
    format: uuid
```

**Step 2: Update Confbridge schema (same change)**

**Step 3: Regenerate OpenAPI types**

```bash
cd bin-openapi-manager && go generate ./...
cd bin-api-manager && go generate ./...
```

**Step 4: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-external-media-ids-array

- bin-openapi-manager: Update Call and Confbridge schemas from external_media_id to external_media_ids array
- bin-api-manager: Regenerate server code for updated OpenAPI spec"
```

---

### Task 15: Run Full Verification Workflow

**Step 1: Verify bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-call-manager**

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Check for any other services that import updated requesthandler functions**

```bash
grep -r "CallV1CallExternalMediaStop\|CallV1ConfbridgeExternalMediaStop" --include="*.go" -l | grep -v vendor | grep -v mock_ | grep -v _test.go
```

Verify and fix each service found.

**Step 6: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "NOJIRA-external-media-ids-array

- Fix verification issues across all affected services"
```

---

## Summary of Changes by Service

| Service | Changes |
|---------|---------|
| **bin-call-manager** | Model field changes, DB handler add/remove ops, call/confbridge handler updates, external media handler parent cleanup, listen handler body parsing, action handler array iteration, test SQL schema |
| **bin-common-handler** | RequestHandler interface and implementation — add externalMediaID param to stop functions |
| **bin-dbscheme-manager** | Alembic migration |
| **bin-openapi-manager** | OpenAPI schema update |
| **bin-api-manager** | Regenerate server code |

## Services NOT Changed

- `bin-tts-manager` — uses `CallV1ExternalMediaStop(ctx, id)` (direct external media path, unchanged)
- `bin-transcribe-manager` — same
- `bin-pipecat-manager` — same
