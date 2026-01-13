package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	rmroute "monorepo/bin-route-manager/models/route"

	"monorepo/bin-call-manager/models/call"
)

var (
	callTable = "call_calls"
)

// callGetFromRow gets the call from the row.
func (h *handler) callGetFromRow(row *sql.Rows) (*call.Call, error) {
	res := &call.Call{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. callGetFromRow. err: %v", err)
	}

	// Initialize nil slices/maps to empty
	if res.ChainedCallIDs == nil {
		res.ChainedCallIDs = []uuid.UUID{}
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}
	if res.Data == nil {
		res.Data = map[call.DataType]string{}
	}
	if res.Dialroutes == nil {
		res.Dialroutes = []rmroute.Route{}
	}

	return res, nil
}

// CallCreate creates new call record.
func (h *handler) CallCreate(ctx context.Context, c *call.Call) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = commondatabasehandler.DefaultTimeStamp
	c.TMDelete = commondatabasehandler.DefaultTimeStamp
	c.TMRinging = commondatabasehandler.DefaultTimeStamp
	c.TMProgressing = commondatabasehandler.DefaultTimeStamp
	c.TMHangup = commondatabasehandler.DefaultTimeStamp

	// Initialize nil slices
	if c.ChainedCallIDs == nil {
		c.ChainedCallIDs = []uuid.UUID{}
	}
	if c.RecordingIDs == nil {
		c.RecordingIDs = []uuid.UUID{}
	}
	if c.Dialroutes == nil {
		c.Dialroutes = []rmroute.Route{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CallCreate. err: %v", err)
	}

	// Add source_target and destination_target for indexed search
	fields["source_target"] = c.Source.Target
	fields["destination_target"] = c.Destination.Target

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(callTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. CallCreate. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, c.ID)

	return nil
}

// CallGet returns call.
func (h *handler) CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	res, err := h.callGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.callGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.callSetToCache(ctx, res)

	return res, nil
}

// CallGetByChannelID returns call by channelID.
func (h *handler) CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error) {
	fields := commondatabasehandler.GetDBFields(&call.Call{})
	query, args, err := squirrel.
		Select(fields...).
		From(callTable).
		Where(squirrel.Eq{string(call.FieldChannelID): channelID}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. CallGetByChannelID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGetByChannelID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. CallGetByChannelID. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. CallGetByChannelID, err: %v", err)
	}

	return res, nil
}

// CallGets returns a list of calls.
func (h *handler) CallGets(ctx context.Context, size uint64, token string, filters map[call.Field]any) ([]*call.Call, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	dbFields := commondatabasehandler.GetDBFields(&call.Call{})
	sb := squirrel.
		Select(dbFields...).
		From(callTable).
		Where(squirrel.Lt{string(call.FieldTMCreate): token}).
		OrderBy(string(call.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. CallGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CallGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*call.Call{}
	for rows.Next() {
		u, err := h.callGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. CallGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. CallGets. err: %v", err)
	}

	return res, nil
}

// CallUpdate updates call fields using a generic typed field map
func (h *handler) CallUpdate(ctx context.Context, id uuid.UUID, fields map[call.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	// Only set TMUpdate if it's not already provided
	if _, ok := fields[call.FieldTMUpdate]; !ok {
		fields[call.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(callTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(call.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("CallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("CallUpdate: exec failed: %w", err)
	}

	_ = h.callUpdateToCache(ctx, id)
	return nil
}

// CallSetBridgeID sets the call bridge id
func (h *handler) CallSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldBridgeID: bridgeID,
	})
}

// CallSetStatusRinging sets the call status to ringing
func (h *handler) CallSetStatusRinging(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldStatus:    call.StatusRinging,
		call.FieldTMRinging: ts,
		call.FieldTMUpdate:  ts,
	})
}

// CallSetStatusProgressing sets the call status to progressing
func (h *handler) CallSetStatusProgressing(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldStatus:        call.StatusProgressing,
		call.FieldTMProgressing: ts,
		call.FieldTMUpdate:      ts,
	})
}

// CallSetStatus sets the call status without update the timestamp for status
func (h *handler) CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldStatus: status,
	})
}

// CallSetHangup sets the call status to hangup
func (h *handler) CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy) error {
	ts := h.utilHandler.TimeGetCurTime()
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldStatus:       call.StatusHangup,
		call.FieldHangupBy:     hangupBy,
		call.FieldHangupReason: reason,
		call.FieldTMHangup:     ts,
		call.FieldTMUpdate:     ts,
	})
}

// CallSetFlowID sets the call's flow_id
func (h *handler) CallSetFlowID(ctx context.Context, id, flowID uuid.UUID) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldFlowID: flowID,
	})
}

// CallSetConfbridgeID sets the call's confbridge_id
func (h *handler) CallSetConfbridgeID(ctx context.Context, id, confbridgeID uuid.UUID) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldConfbridgeID: confbridgeID,
	})
}

// CallSetActionAndActionNextHold sets the call action and action_next_hold
func (h *handler) CallSetActionAndActionNextHold(ctx context.Context, id uuid.UUID, action *fmaction.Action, hold bool) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldAction:         action,
		call.FieldActionNextHold: hold,
	})
}

// callGetFromCache returns call from the cache.
func (h *handler) callGetFromCache(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	res, err := h.cache.CallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// callGetFromDB returns call from the DB.
func (h *handler) callGetFromDB(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	fields := commondatabasehandler.GetDBFields(&call.Call{})
	query, args, err := squirrel.
		Select(fields...).
		From(callTable).
		Where(squirrel.Eq{string(call.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. callGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. callGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. callGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. callGetFromDB. id: %s", id)
	}

	return res, nil
}

// callUpdateToCache gets the call from the DB and update the cache.
func (h *handler) callUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.callGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.callSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// callSetToCache sets the given call to the cache
func (h *handler) callSetToCache(ctx context.Context, c *call.Call) error {
	if err := h.cache.CallSet(ctx, c); err != nil {
		return err
	}
	return nil
}

// CallAddChainedCallID adds the call id to the given call's chained_call_ids.
func (h *handler) CallAddChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		chained_call_ids = json_array_append(
			chained_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallRemoveChainedCallID removes the call id from the given call's chained_call_ids.
func (h *handler) CallRemoveChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		chained_call_ids = json_remove(
			chained_call_ids, replace(
				json_search(
					chained_call_ids,
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

	_, err := h.db.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetMasterCallID sets the call's master_call_id
func (h *handler) CallSetMasterCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldMasterCallID: callID,
	})
}

// CallSetRecordingID sets the given recordID to recording_id.
func (h *handler) CallSetRecordingID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldRecordingID: recordID,
	})
}

// CallSetExternalMediaID sets the call's external_media_id
func (h *handler) CallSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldExternalMediaID: externalMediaID,
	})
}

// CallSetForRouteFailover sets the call for route failover.
func (h *handler) CallSetForRouteFailover(ctx context.Context, id uuid.UUID, channelID string, dialrouteID uuid.UUID) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldBridgeID:    "",
		call.FieldChannelID:   channelID,
		call.FieldDialrouteID: dialrouteID,
	})
}

// CallAddRecordingIDs adds the given recording_id into the recording_ids.
func (h *handler) CallAddRecordingIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		recording_ids = json_array_append(
			recording_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddRecordingIDs. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

func (h *handler) CallTXStart(id uuid.UUID) (*sql.Tx, *call.Call, error) {
	tx, err := h.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get transaction. CallTXStart. err: %v", err)
	}

	fields := commondatabasehandler.GetDBFields(&call.Call{})
	sb := squirrel.
		Select(fields...).
		From(callTable).
		Where(squirrel.Eq{string(call.FieldID): id.Bytes()}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("could not build query. CallTXStart. err: %v", err)
	}

	row, err := tx.Query(query, args...)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("could not query. CallTXStart. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		_ = tx.Rollback()
		return nil, nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("could not get call. CallTXStart, err: %v", err)
	}

	return tx, res, nil
}

func (h *handler) CallTXFinish(tx *sql.Tx, commit bool) {
	if commit {
		_ = tx.Commit()
	} else {
		_ = tx.Rollback()
	}
}

// CallTXAddChainedCallID adds the call id to the given call's chained_call_ids in a transaction mode.
func (h *handler) CallTXAddChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		chained_call_ids = json_array_append(
			chained_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := tx.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(context.Background(), id)

	return nil
}

// CallTXRemoveChainedCallID removes the call id from the given call's chained_call_ids in a transaction mode.
func (h *handler) CallTXRemoveChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		chained_call_ids = json_remove(
			chained_call_ids, replace(
				json_search(
					chained_call_ids,
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

	_, err := tx.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(context.Background(), id)

	return nil
}

// CallSetActionNextHold sets the action_next_hold.
func (h *handler) CallSetActionNextHold(ctx context.Context, id uuid.UUID, hold bool) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldActionNextHold: hold,
	})
}

// CallDelete deletes the call
func (h *handler) CallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[call.Field]any{
		call.FieldTMUpdate: ts,
		call.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CallDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(callTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(call.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("CallDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("CallDelete: exec failed: %w", err)
	}

	_ = h.callUpdateToCache(ctx, id)
	return nil
}

// CallSetData sets the call data
func (h *handler) CallSetData(ctx context.Context, id uuid.UUID, data map[call.DataType]string) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldData: data,
	})
}

// CallSetMuteDirection sets the call mute direction
func (h *handler) CallSetMuteDirection(ctx context.Context, id uuid.UUID, muteDirection call.MuteDirection) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldMuteDirection: muteDirection,
	})
}

// CallSetChannelIDAndBridgeID sets the call's channel_id and bridge_id
func (h *handler) CallSetChannelIDAndBridgeID(ctx context.Context, id uuid.UUID, channelID string, bridgeID string) error {
	return h.CallUpdate(ctx, id, map[call.Field]any{
		call.FieldChannelID: channelID,
		call.FieldBridgeID:  bridgeID,
	})
}
