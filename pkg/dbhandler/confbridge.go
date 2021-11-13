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
		conference_id,
		bridge_id,

		channel_call_ids,

		recording_id,
		recording_ids,

		tm_create,
		tm_update,
		tm_delete

	from
		confbridges
	`
)

// confbridgeGetFromRow gets the confbridge from the row.
func (h *handler) confbridgeGetFromRow(row *sql.Rows) (*confbridge.Confbridge, error) {

	var recordingIDs string
	var channelCallIDs string

	res := &confbridge.Confbridge{}
	if err := row.Scan(
		&res.ID,
		&res.ConferenceID,
		&res.BridgeID,

		&channelCallIDs,

		&res.RecordingID,
		&recordingIDs,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. confbridgeGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(channelCallIDs), &res.ChannelCallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the callChannelIDs. confbridgeGetFromRow. err: %v", err)
	}
	if res.ChannelCallIDs == nil {
		res.ChannelCallIDs = map[string]uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(recordingIDs), &res.RecordingIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the recordingIDs. confbridgeGetFromRow. err: %v", err)
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}

	return res, nil
}

// ConfbridgeGetFromCache returns conference from the cache if possible.
func (h *handler) ConfbridgeGetFromCache(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {

	// get from cache
	res, err := h.cache.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ConfbridgeGetFromDB gets confbridge.
func (h *handler) ConfbridgeGetFromDB(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {

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

// ConfbridgeUpdateToCache gets the confbridge from the DB and update the cache.
func (h *handler) ConfbridgeUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.ConfbridgeGetFromDB(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get confbridge from the DB. err: %v", err)
		return err
	}

	if err := h.ConfbridgeSetToCache(ctx, res); err != nil {
		logrus.Errorf("Could not set confbridge to the cache. err: %v", err)
		return err
	}

	return nil
}

// ConfbridgeSetToCache sets the given conference to the cache
func (h *handler) ConfbridgeSetToCache(ctx context.Context, data *confbridge.Confbridge) error {
	if err := h.cache.ConfbridgeSet(ctx, data); err != nil {
		return err
	}

	return nil
}

// ConfbridgeCreate creates a new conference record.
func (h *handler) ConfbridgeCreate(ctx context.Context, cb *confbridge.Confbridge) error {
	q := `insert into confbridges(
		id,
		conference_id,
		bridge_id,

		channel_call_ids,

		recording_id,
		recording_ids,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?,
		?, ?,
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
		cb.ConferenceID.Bytes(),
		cb.BridgeID,

		callChannelIDs,

		cb.RecordingID.Bytes(),
		recordingIDs,

		cb.TMCreate,
		cb.TMUpdate,
		cb.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeCreate. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, cb.ID)

	return nil
}

// ConfbridgeGet gets conference.
func (h *handler) ConfbridgeGet(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {

	res, err := h.ConfbridgeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.ConfbridgeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.ConfbridgeSetToCache(ctx, res)

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

	_, err := h.db.Exec(q, bridgeID, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetBridgeID. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, id)

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

	_, err := h.db.Exec(q, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeDelete. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeSetRecordID sets the conference's recording_id.
func (h *handler) ConfbridgeSetRecordID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	// prepare
	q := `
	update confbridges set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeSetRecordID. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, id)

	return nil
}

// ConfbridgeAddRecordIDs adds the record file to the bridge's record_files.
func (h *handler) ConfbridgeAddRecordIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
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

	_, err := h.db.Exec(q, recordID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeAddRecordIDs. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, id)

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

	_, err := h.db.Exec(q, key, callID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeAddChannelCallID. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, id)

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

	_, err := h.db.Exec(q, key, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConfbridgeRemoveChannelCallID. err: %v", err)
	}

	// update the cache
	_ = h.ConfbridgeUpdateToCache(ctx, id)

	return nil
}
