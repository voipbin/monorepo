package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
)

const (
	bridgeSelect = `
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

		reference_type,
		reference_id,

		tm_create,
		tm_update,
		tm_delete
	from
		bridges
	`
)

func (h *handler) bridgeGetFromRow(row *sql.Rows) (*bridge.Bridge, error) {
	var channelIDs sql.NullString

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

		&res.ReferenceType,
		&res.ReferenceID,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. bridgeGetFromRow. err: %v", err)
	}

	if channelIDs.Valid {
		if err := json.Unmarshal([]byte(channelIDs.String), &res.ChannelIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the channel_ids. bridgeGetFromRow. err: %v", err)
		}
	}
	if res.ChannelIDs == nil {
		res.ChannelIDs = []string{}
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

		reference_type,
		reference_id,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?,
		?, ?,
		?, ?, ?
		)
	`

	tmpChannelIDs, err := json.Marshal(b.ChannelIDs)
	if err != nil {
		return fmt.Errorf("could not marshal channel_ids. BridgeCreate. err: %v", err)
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

		b.ReferenceType,
		b.ReferenceID.Bytes(),

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeCreate. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, b.ID)

	return nil
}

// bridgeGetFromDB returns bridge from the DB.
func (h *handler) bridgeGetFromDB(ctx context.Context, id string) (*bridge.Bridge, error) {

	q := fmt.Sprintf("%s where id = ?", bridgeSelect)

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. bridgeGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.bridgeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. bridgeGetFromDB. err: %v", err)
	}

	return res, nil
}

// bridgeUpdateToCache gets the bridge from the DB and update the cache.
func (h *handler) bridgeUpdateToCache(ctx context.Context, id string) error {

	res, err := h.bridgeGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.bridgeSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// bridgeSetToCache sets the given bridge to the cache
func (h *handler) bridgeSetToCache(ctx context.Context, bridge *bridge.Bridge) error {
	if err := h.cache.BridgeSet(ctx, bridge); err != nil {
		return err
	}

	return nil
}

// bridgeGetFromCache returns bridge from the cache.
func (h *handler) bridgeGetFromCache(ctx context.Context, id string) (*bridge.Bridge, error) {

	// get from cache
	res, err := h.cache.BridgeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// BridgeGet returns bridge.
func (h *handler) BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error) {

	res, err := h.bridgeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.bridgeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.bridgeSetToCache(ctx, res)

	return res, nil
}

// BridgeEnd updates the bridge end.
func (h *handler) BridgeEnd(ctx context.Context, id string) error {
	// prepare
	q := `
	update bridges set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeEnd. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, id)

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

	_, err := h.db.Exec(q, channelID, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeAddChannelID. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, id)

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

	_, err := h.db.Exec(q, channelID, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeRemoveChannelID. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, id)

	return nil
}
