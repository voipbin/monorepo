package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/confbridge"
)

const (
	confbridgeSelect = `
	select
		id,
		customer_id,

		activeflow_id,
		reference_type,
		reference_id,

		type,
		status,
		bridge_id,
		flags,

		channel_call_ids,

		recording_id,
		recording_ids,

		external_media_id,

		tm_create,
		tm_update,
		tm_delete

	from
		call_confbridges
	`
)

// confbridgeGetFromRow gets the confbridge from the row.
func (h *handler) confbridgeGetFromRow(row *sql.Rows) (*confbridge.Confbridge, error) {
	var recordingIDs sql.NullString
	var channelCallIDs sql.NullString
	var flags sql.NullString

	res := &confbridge.Confbridge{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ActiveflowID,
		&res.ReferenceType,
		&res.ReferenceID,

		&res.Type,
		&res.Status,
		&res.BridgeID,
		&flags,

		&channelCallIDs,

		&res.RecordingID,
		&recordingIDs,

		&res.ExternalMediaID,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. confbridgeGetFromRow. err: %v", err)
	}

	// ChannelCallIDs
	if channelCallIDs.Valid {
		if err := json.Unmarshal([]byte(channelCallIDs.String), &res.ChannelCallIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the callChannelIDs. confbridgeGetFromRow. err: %v", err)
		}
	}
	if res.ChannelCallIDs == nil {
		res.ChannelCallIDs = map[string]uuid.UUID{}
	}

	// RecordingIDs
	if recordingIDs.Valid {
		if err := json.Unmarshal([]byte(recordingIDs.String), &res.RecordingIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the recordingIDs. confbridgeGetFromRow. err: %v", err)
		}
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}

	if flags.Valid {
		if err := json.Unmarshal([]byte(flags.String), &res.Flags); err != nil {
			return nil, fmt.Errorf("could not unmarshal the flags. confbridgeGetFromRow. err: %v", err)
		}
	}
	if res.Flags == nil {
		res.Flags = []confbridge.Flag{}
	}

	return res, nil
}

// ConfbridgeCreate creates a new confbridge record.
func (h *handler) ConfbridgeCreate(ctx context.Context, cb *confbridge.Confbridge) error {
	q := `insert into call_confbridges(
		id,
		customer_id,

		activeflow_id,
		reference_type,
		reference_id,

		type,
		status,
		bridge_id,
		flags,

		channel_call_ids,

		recording_id,
		recording_ids,

		external_media_id,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?, ?,
		?, ?, ?, ?,
		?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	flags, err := json.Marshal(cb.Flags)
	if err != nil {
		return fmt.Errorf("could not marshal flags. ConfbridgeCreate. err: %v", err)
	}

	callChannelIDs, err := json.Marshal(cb.ChannelCallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal callChannelIDs. ConfbridgeCreate. err: %v", err)
	}

	recordingIDs, err := json.Marshal(cb.RecordingIDs)
	if err != nil {
		return fmt.Errorf("could not marshal recording_ids. ConfbridgeCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cb.ID.Bytes(),
		cb.CustomerID.Bytes(),

		cb.ActiveflowID.Bytes(),
		cb.ReferenceType,
		cb.ReferenceID.Bytes(),

		cb.Type,
		cb.Status,
		cb.BridgeID,
		flags,

		callChannelIDs,

		cb.RecordingID.Bytes(),
		recordingIDs,

		cb.ExternalMediaID.Bytes(),

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeCreate. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, cb.ID)

	return nil
}

// confbridgeGetFromCache returns conference from the cache if possible.
func (h *handler) confbridgeGetFromCache(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {

	// get from cache
	res, err := h.cache.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// confbridgeGetFromDB gets confbridge.
func (h *handler) confbridgeGetFromDB(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", confbridgeSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfbridgeGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
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

	// prepare
	q := fmt.Sprintf("%s where bridge_id = ?", confbridgeSelect)

	row, err := h.db.Query(q, bridgeID)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfbridgeGetByBridgeID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.confbridgeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConfbridgeGetByBridgeID, err: %v", err)
	}

	return res, nil
}

// ConfbridgeGets returns a list of confbridges.
func (h *handler) ConfbridgeGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*confbridge.Confbridge, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, confbridgeSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "activeflow_id", "reference_id", "recording_id", "external_media_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfbridgeGets. err: %v", err)
	}
	defer rows.Close()

	res := []*confbridge.Confbridge{}
	for rows.Next() {
		u, err := h.confbridgeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. confbridgeGetFromRow, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ConfbridgeSetBridgeID sets the bridge id
func (h *handler) ConfbridgeSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error {
	//prepare
	q := `
	update call_confbridges set
		bridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, bridgeID, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetBridgeID. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeDelete ends the conference
func (h *handler) ConfbridgeDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update call_confbridges set
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeDelete. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeSetRecordingID sets the conference's recording_id.
func (h *handler) ConfbridgeSetRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	// prepare
	q := `
	update call_confbridges set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordingID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetRecordingID. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeAddRecordingIDs adds the record file to the bridge's record_files.
func (h *handler) ConfbridgeAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	// prepare
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

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeSetExternalMediaID sets the conference's external media id.
func (h *handler) ConfbridgeSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error {
	// prepare
	q := `
	update call_confbridges set
		external_media_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetExternalMediaID. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
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

	// update the cache
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

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeSetFlags sets the confbridge's flags
func (h *handler) ConfbridgeSetFlags(ctx context.Context, id uuid.UUID, flags []confbridge.Flag) error {

	// prepare
	q := `
	update
		call_confbridges
	set
		flags = ?,
		tm_update = ?
	where
		id = ?
	`
	tmp, err := json.Marshal(flags)
	if err != nil {
		return errors.Wrap(err, "could not marshal the flags")
	}

	_, err = h.db.Exec(q, tmp, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetFlags. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeSetStatus sets the status
func (h *handler) ConfbridgeSetStatus(ctx context.Context, id uuid.UUID, status confbridge.Status) error {
	//prepare
	q := `
	update call_confbridges set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}
