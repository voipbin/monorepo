package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
)

// BridgeUpdateToCache gets the bridge from the DB and update the cache.
func (h *handler) BridgeUpdateToCache(ctx context.Context, id string) error {

	res, err := h.BridgeGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.BridgeSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// BridgeSetToCache sets the given bridge to the cache
func (h *handler) BridgeSetToCache(ctx context.Context, bridge *bridge.Bridge) error {
	if err := h.cache.BridgeSet(ctx, bridge); err != nil {
		return err
	}

	return nil
}

// BridgeGetFromCache returns bridge from the cache.
func (h *handler) BridgeGetFromCache(ctx context.Context, id string) (*bridge.Bridge, error) {

	// get from cache
	res, err := h.cache.BridgeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) BridgeCreate(ctx context.Context, b *bridge.Bridge) error {
	q := `insert into bridges(
		id,
		asterisk_id,
		name,

		type,
		tech,
		class,
		creator,

		video_mode,
		video_source_id,

		channel_ids,

		record_channel_id,
		record_files,

		conference_id,
		conference_type,
		conference_join,

		tm_create
	) values(
		?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?,
		?, ?,
		?, ?, ?,
		?
		)
	`

	tmpChannelIDs, err := json.Marshal(b.ChannelIDs)
	if err != nil {
		return fmt.Errorf("could not marshal channel_ids. BridgeCreate. err: %v", err)
	}

	tmpRecordFiles, err := json.Marshal(b.RecordFiles)
	if err != nil {
		return fmt.Errorf("could not marshal record_files. BridgeCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		b.ID,
		b.AsteriskID,
		b.Name,

		b.Type,
		b.Tech,
		b.Class,
		b.Creator,

		b.VideoMode,
		b.VideoSourceID,

		tmpChannelIDs,

		b.RecordChannelID,
		tmpRecordFiles,

		b.ConferenceID.Bytes(),
		b.ConferenceType,
		b.ConferenceJoin,

		b.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeCreate. err: %v", err)
	}

	// update the cache
	h.BridgeUpdateToCache(ctx, b.ID)

	return nil
}

// BridgeGet returns bridge.
func (h *handler) BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error) {

	res, err := h.BridgeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.BridgeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.BridgeSetToCache(ctx, res)

	return res, nil
}

// BridgeEnd updates the bridge end.
func (h *handler) BridgeEnd(ctx context.Context, id, timestamp string) error {
	// prepare
	q := `
	update bridges set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, getCurTime(), timestamp, id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeEnd. err: %v", err)
	}

	// update the cache
	h.BridgeUpdateToCache(ctx, id)

	return nil
}

// BridgeAddChannel adds the channel to the bridge.
func (h *handler) BridgeAddChannelID(ctx context.Context, id, channelID string) error {
	// prepare
	q := `
	update bridges set
		channel_ids = json_array_append(
			channel_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, channelID, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeAddChannelID. err: %v", err)
	}

	// update the cache
	h.BridgeUpdateToCache(ctx, id)

	return nil
}

// BridgeRemoveChannelID removes the channel from the bridge.
func (h *handler) BridgeRemoveChannelID(ctx context.Context, id, channelID string) error {
	// prepare
	q := `
	update bridges set
		channel_ids = json_remove(
			channel_ids, replace(
				json_search(
					channel_ids,
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

	_, err := h.db.Exec(q, channelID, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeRemoveChannelID. err: %v", err)
	}

	// update the cache
	h.BridgeUpdateToCache(ctx, id)

	return nil
}

// BridgeGetUntilTimeout gets the bridge until the ctx is timed out.
func (h *handler) BridgeGetUntilTimeout(ctx context.Context, id string) (*bridge.Bridge, error) {

	chanRes := make(chan *bridge.Bridge)
	stop := false

	go func() {
		for {
			if stop == true {
				return
			}

			tmp, err := h.BridgeGet(ctx, id)
			if err != nil {
				time.Sleep(defaultDelayTimeout)
				continue
			}

			chanRes <- tmp
			return
		}
	}()

	select {
	case res := <-chanRes:
		return res, nil
	case <-ctx.Done():
		stop = true
		return nil, fmt.Errorf("could not get bridge. err: tiemout")
	}
}

// BridgeIsExist returns true if the bridge exist within timeout.
func (h *handler) BridgeIsExist(id string, timeout time.Duration) bool {
	// check the channel is exists
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := h.BridgeGetUntilTimeout(ctx, id)
	if err != nil {
		return false
	}
	return true
}

// BridgeGetFromDB returns bridge from the DB.
func (h *handler) BridgeGetFromDB(ctx context.Context, id string) (*bridge.Bridge, error) {

	// prepare
	q := `
	select
		id,
		asterisk_id,
		name,

		type,
		tech,
		class,
		creator,

		video_mode,
		video_source_id,

		channel_ids,

		record_channel_id,
		record_files,

		conference_id,
		conference_type,
		conference_join,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		bridges
	where
		id = ?
	`

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. BridgeGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	var channelIDs string
	var recordFiles string
	res := &bridge.Bridge{}
	if err := row.Scan(
		&res.ID,
		&res.AsteriskID,
		&res.Name,

		&res.Type,
		&res.Tech,
		&res.Class,
		&res.Creator,

		&res.VideoMode,
		&res.VideoSourceID,

		&channelIDs,

		&res.RecordChannelID,
		&recordFiles,

		&res.ConferenceID,
		&res.ConferenceType,
		&res.ConferenceJoin,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. BridgeGetFromDB. err: %v", err)
	}

	if err := json.Unmarshal([]byte(channelIDs), &res.ChannelIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the channel_ids. BridgeGetFromDB. err: %v", err)
	}
	if res.ChannelIDs == nil {
		res.ChannelIDs = []string{}
	}

	if err := json.Unmarshal([]byte(recordFiles), &res.RecordFiles); err != nil {
		return nil, fmt.Errorf("could not unmarshal the record_files. BridgeGetFromDB. err: %v", err)
	}
	if res.RecordFiles == nil {
		res.RecordFiles = []string{}
	}

	return res, nil
}

// BridgeSetRecordChannelID sets the bridge's record_channel_id.
func (h *handler) BridgeSetRecordChannelID(ctx context.Context, id, recordChannelID string) error {
	// prepare
	q := `
	update bridges set
		record_channel_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordChannelID, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeSetRecordChannelID. err: %v", err)
	}

	// update the cache
	h.BridgeUpdateToCache(ctx, id)

	return nil
}

// BridgeAddRecordFiles adds the record file to the bridge's record_files.
func (h *handler) BridgeAddRecordFiles(ctx context.Context, id, filename string) error {
	// prepare
	q := `
	update bridges set
		record_files = json_array_append(
			record_files,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, filename, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeAddRecordFiles. err: %v", err)
	}

	// update the cache
	h.BridgeUpdateToCache(ctx, id)

	return nil
}
