package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

const (
	confbridgeSelect = `
	select
		id,
		customer_id,
		type,
		bridge_id,

		channel_call_ids,

		recording_id,
		recording_ids,

		external_media_id,

		tm_create,
		tm_update,
		tm_delete

	from
		confbridges
	`
)

// confbridgeGetFromRow gets the confbridge from the row.
func (h *handler) confbridgeGetFromRow(row *sql.Rows) (*confbridge.Confbridge, error) {
	var recordingIDs sql.NullString
	var channelCallIDs sql.NullString

	res := &confbridge.Confbridge{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Type,
		&res.BridgeID,

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

	return res, nil
}

// ConfbridgeCreate creates a new confbridge record.
func (h *handler) ConfbridgeCreate(ctx context.Context, cb *confbridge.Confbridge) error {
	q := `insert into confbridges(
		id,
		customer_id,
		type,
		bridge_id,

		channel_call_ids,

		recording_id,
		recording_ids,

		external_media_id,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?,
		?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	callChannelIDs, err := json.Marshal(cb.ChannelCallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ConfbridgeCreate. err: %v", err)
	}

	recordingIDs, err := json.Marshal(cb.RecordingIDs)
	if err != nil {
		return fmt.Errorf("could not marshal recording_ids. ConfbridgeCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cb.ID.Bytes(),
		cb.CustomerID.Bytes(),
		cb.Type,
		cb.BridgeID,

		callChannelIDs,

		cb.RecordingID.Bytes(),
		recordingIDs,

		cb.ExternalMediaID.Bytes(),

		h.utilHandler.GetCurTime(),
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

// ConfbridgeSetBridgeID sets the bridge id
func (h *handler) ConfbridgeSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error {
	//prepare
	q := `
	update confbridges set
		bridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, bridgeID, h.utilHandler.GetCurTime(), id.Bytes())
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
	update confbridges set
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, h.utilHandler.GetCurTime(), id.Bytes())
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
	update confbridges set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordingID.Bytes(), h.utilHandler.GetCurTime(), id.Bytes())
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
	update confbridges set
		recording_ids = json_array_append(
			recording_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordingID.Bytes(), h.utilHandler.GetCurTime(), id.Bytes())
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
	update confbridges set
		external_media_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.Bytes(), h.utilHandler.GetCurTime(), id.Bytes())
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
	update confbridges set
		channel_call_ids = json_insert(
			channel_call_ids,
			?,
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, key, callID.String(), h.utilHandler.GetCurTime(), id.Bytes())
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
	update confbridges set
		channel_call_ids = json_remove(
			channel_call_ids,
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, key, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeRemoveChannelCallID. err: %v", err)
	}

	// update the cache
	_ = h.confbridgeUpdateToCache(ctx, id)

	return nil
}
