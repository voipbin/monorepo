package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/confbridge"
)

var (
	confbridgeTable = "call_confbridges"
)

// confbridgeGetFromRow gets the confbridge from the row.
func (h *handler) confbridgeGetFromRow(row *sql.Rows) (*confbridge.Confbridge, error) {
	res := &confbridge.Confbridge{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. confbridgeGetFromRow. err: %v", err)
	}

	// Initialize nil slices/maps to empty
	if res.ChannelCallIDs == nil {
		res.ChannelCallIDs = map[string]uuid.UUID{}
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}
	if res.Flags == nil {
		res.Flags = []confbridge.Flag{}
	}

	return res, nil
}

// ConfbridgeCreate creates a new confbridge record.
func (h *handler) ConfbridgeCreate(ctx context.Context, cb *confbridge.Confbridge) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	cb.TMCreate = now
	cb.TMUpdate = commondatabasehandler.DefaultTimeStamp
	cb.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Initialize nil slices/maps
	if cb.Flags == nil {
		cb.Flags = []confbridge.Flag{}
	}
	if cb.ChannelCallIDs == nil {
		cb.ChannelCallIDs = map[string]uuid.UUID{}
	}
	if cb.RecordingIDs == nil {
		cb.RecordingIDs = []uuid.UUID{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(cb)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ConfbridgeCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(confbridgeTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ConfbridgeCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. ConfbridgeCreate. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, cb.ID)

	return nil
}

// confbridgeGetFromCache returns conference from the cache if possible.
func (h *handler) confbridgeGetFromCache(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	res, err := h.cache.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// confbridgeGetFromDB gets confbridge.
func (h *handler) confbridgeGetFromDB(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	fields := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})
	query, args, err := squirrel.
		Select(fields...).
		From(confbridgeTable).
		Where(squirrel.Eq{string(confbridge.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. confbridgeGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfbridgeGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. confbridgeGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.confbridgeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get confbridge. ConfbridgeGetFromDB, err: %v", err)
	}

	return res, nil
}

// confbridgeUpdateToCache gets the confbridge from the DB and update the cache.
func (h *handler) confbridgeUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.confbridgeGetFromDB(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get confbridge from the DB. err: %v", err)
		return err
	}

	if err := h.confbridgeSetToCache(ctx, res); err != nil {
		logrus.Errorf("Could not set confbridge to the cache. err: %v", err)
		return err
	}

	return nil
}

// confbridgeSetToCache sets the given conference to the cache
func (h *handler) confbridgeSetToCache(ctx context.Context, data *confbridge.Confbridge) error {
	if err := h.cache.ConfbridgeSet(ctx, data); err != nil {
		return err
	}
	return nil
}

// ConfbridgeGet gets conference.
func (h *handler) ConfbridgeGet(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	res, err := h.confbridgeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.confbridgeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.confbridgeSetToCache(ctx, res)

	return res, nil
}

// ConfbridgeGetByBridgeID gets confbridge by the bridgeID.
func (h *handler) ConfbridgeGetByBridgeID(ctx context.Context, bridgeID string) (*confbridge.Confbridge, error) {
	fields := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})
	query, args, err := squirrel.
		Select(fields...).
		From(confbridgeTable).
		Where(squirrel.Eq{string(confbridge.FieldBridgeID): bridgeID}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ConfbridgeGetByBridgeID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfbridgeGetByBridgeID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. ConfbridgeGetByBridgeID. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.confbridgeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConfbridgeGetByBridgeID, err: %v", err)
	}

	return res, nil
}

// ConfbridgeGets returns a list of confbridges.
func (h *handler) ConfbridgeList(ctx context.Context, size uint64, token string, filters map[confbridge.Field]any) ([]*confbridge.Confbridge, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	dbFields := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})
	sb := squirrel.
		Select(dbFields...).
		From(confbridgeTable).
		Where(squirrel.Lt{string(confbridge.FieldTMCreate): token}).
		OrderBy(string(confbridge.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ConfbridgeGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ConfbridgeGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfbridgeGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*confbridge.Confbridge{}
	for rows.Next() {
		u, err := h.confbridgeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. confbridgeGetFromRow, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ConfbridgeGets. err: %v", err)
	}

	return res, nil
}

// ConfbridgeUpdate updates confbridge fields using a generic typed field map
func (h *handler) ConfbridgeUpdate(ctx context.Context, id uuid.UUID, fields map[confbridge.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[confbridge.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ConfbridgeUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(confbridgeTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(confbridge.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ConfbridgeUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ConfbridgeUpdate: exec failed: %w", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}

// ConfbridgeSetBridgeID sets the bridge id
func (h *handler) ConfbridgeSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error {
	return h.ConfbridgeUpdate(ctx, id, map[confbridge.Field]any{
		confbridge.FieldBridgeID: bridgeID,
	})
}

// ConfbridgeDelete ends the conference
func (h *handler) ConfbridgeDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()
	return h.ConfbridgeUpdate(ctx, id, map[confbridge.Field]any{
		confbridge.FieldTMDelete: ts,
	})
}

// ConfbridgeSetRecordingID sets the conference's recording_id.
func (h *handler) ConfbridgeSetRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	return h.ConfbridgeUpdate(ctx, id, map[confbridge.Field]any{
		confbridge.FieldRecordingID: recordingID,
	})
}

// ConfbridgeAddRecordingIDs adds the record file to the bridge's record_files.
func (h *handler) ConfbridgeAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	// Use raw SQL for JSON array append operation
	q := `
	update call_confbridges set
		recording_ids = json_array_append(
			recording_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordingID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeAddRecordingIDs. err: %v", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}

// ConfbridgeSetExternalMediaID sets the conference's external media id.
func (h *handler) ConfbridgeSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error {
	return h.ConfbridgeUpdate(ctx, id, map[confbridge.Field]any{
		confbridge.FieldExternalMediaID: externalMediaID,
	})
}

// ConfbridgeAddChannelCallID adds the call/channel id info
func (h *handler) ConfbridgeAddChannelCallID(ctx context.Context, id uuid.UUID, channelID string, callID uuid.UUID) error {
	key := fmt.Sprintf("$.\"%s\"", channelID)
	q := `
	update call_confbridges set
		channel_call_ids = json_insert(
			channel_call_ids,
			?,
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, key, callID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeAddChannelCallID. err: %v", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}

// ConfbridgeRemoveChannelCallID removes the channel/call id info
func (h *handler) ConfbridgeRemoveChannelCallID(ctx context.Context, id uuid.UUID, channelID string) error {
	key := fmt.Sprintf("$.\"%s\"", channelID)
	q := `
	update call_confbridges set
		channel_call_ids = json_remove(
			channel_call_ids,
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, key, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeRemoveChannelCallID. err: %v", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}

// ConfbridgeSetFlags sets the confbridge's flags
func (h *handler) ConfbridgeSetFlags(ctx context.Context, id uuid.UUID, flags []confbridge.Flag) error {
	if flags == nil {
		flags = []confbridge.Flag{}
	}

	tmp, err := json.Marshal(flags)
	if err != nil {
		return errors.Wrap(err, "could not marshal the flags")
	}

	q := `
	update
		call_confbridges
	set
		flags = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err = h.db.Exec(q, tmp, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetFlags. err: %v", err)
	}

	_ = h.confbridgeUpdateToCache(ctx, id)
	return nil
}

// ConfbridgeSetStatus sets the status
func (h *handler) ConfbridgeSetStatus(ctx context.Context, id uuid.UUID, status confbridge.Status) error {
	return h.ConfbridgeUpdate(ctx, id, map[confbridge.Field]any{
		confbridge.FieldStatus: status,
	})
}
