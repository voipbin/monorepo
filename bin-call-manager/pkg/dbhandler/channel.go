package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

const (
	// select query for call get
	channelSelect = `
	select
		asterisk_id,
		id,
		name,
		type,
		tech,

		sip_call_id,
		sip_transport,

		src_name,
		src_number,
		dst_name,
		dst_number,

		state,
		data,
		stasis_name,
		stasis_data,
		bridge_id,
		playback_id,

		dial_result,
		hangup_cause,

		direction,
		mute_direction,

		tm_create,
		tm_update,
		tm_delete,

		tm_answer,
		tm_ringing,
		tm_end

	from
		call_channels
	where
		id = ?
	`
)

// channelGetFromRow gets the channel from the row.
func (h *handler) channelGetFromRow(row *sql.Rows) (*channel.Channel, error) {
	var data sql.NullString
	var stasisData sql.NullString
	var tmDelete sql.NullString

	res := &channel.Channel{}
	if err := row.Scan(
		&res.AsteriskID,
		&res.ID,
		&res.Name,
		&res.Type,
		&res.Tech,

		&res.SIPCallID,
		&res.SIPTransport,

		&res.SourceName,
		&res.SourceNumber,
		&res.DestinationName,
		&res.DestinationNumber,

		&res.State,
		&data,
		&res.StasisName,
		&stasisData,
		&res.BridgeID,
		&res.PlaybackID,

		&res.DialResult,
		&res.HangupCause,

		&res.Direction,
		&res.MuteDirection,

		&res.TMCreate,
		&res.TMUpdate,
		&tmDelete,

		&res.TMAnswer,
		&res.TMRinging,
		&res.TMEnd,
	); err != nil {
		return nil, fmt.Errorf("channelGetFromRow: Could not scan the row. err: %v", err)
	}

	// Data
	if data.Valid {
		if err := json.Unmarshal([]byte(data.String), &res.Data); err != nil {
			return nil, fmt.Errorf("channelGetFromRow: Could not unmarshal the data. err: %v", err)
		}
	}
	if res.Data == nil {
		res.Data = map[string]interface{}{}
	}

	// StasisData
	if stasisData.Valid {
		if err := json.Unmarshal([]byte(stasisData.String), &res.StasisData); err != nil {
			return nil, fmt.Errorf("channelGetFromRow: Could not unmarshal the stasis data. err: %v", err)
		}
	}
	if res.StasisData == nil {
		res.StasisData = map[channel.StasisDataType]string{}
	}

	// TMDelete
	if tmDelete.Valid {
		res.TMDelete = tmDelete.String
	} else {
		res.TMDelete = DefaultTimeStamp
	}

	return res, nil
}

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) ChannelCreate(ctx context.Context, c *channel.Channel) error {
	q := `insert into call_channels(
		asterisk_id,
		id,
		name,
		type,
		tech,

		sip_call_id,
		sip_transport,

		src_name,
		src_number,
		dst_name,
		dst_number,

		state,
		data,
		stasis_name,
		stasis_data,
		bridge_id,
		playback_id,

		dial_result,
		hangup_cause,

		direction,
		mute_direction,

		tm_create,
		tm_update,
		tm_delete,

		tm_answer,
		tm_ringing,
		tm_end

	) values(
		?, ?, ?, ?, ?,
		?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?, ?,
		?, ?,
		?, ?,
		?, ?, ?,
		?, ?, ?
		)
	`

	data, err := json.Marshal(c.Data)
	if err != nil {
		return fmt.Errorf("ChannelCreate: Could not marshal the data. err: %v", err)
	}
	stasisData, err := json.Marshal(c.StasisData)
	if err != nil {
		return fmt.Errorf("ChannelCreate: Could not marshal the stasis data. err: %v", err)
	}

	_, err = h.db.Exec(q,
		c.AsteriskID,
		c.ID,
		c.Name,
		c.Type,
		c.Tech,

		c.SIPCallID,
		c.SIPTransport,

		c.SourceName,
		c.SourceNumber,
		c.DestinationName,
		c.DestinationNumber,

		c.State,
		data,
		c.StasisName,
		stasisData,
		c.BridgeID,
		c.PlaybackID,

		c.DialResult,
		c.HangupCause,

		c.Direction,
		c.MuteDirection,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,

		DefaultTimeStamp,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("ChannelCreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, c.ID)

	return nil
}

// ChannelGet returns channel.
func (h *handler) ChannelGet(ctx context.Context, id string) (*channel.Channel, error) {

	res, err := h.channelGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.channelGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.channelSetToCache(ctx, res)

	return res, nil
}

// ChannelSetData sets the data
func (h *handler) ChannelSetData(ctx context.Context, id string, data map[string]interface{}) error {
	//prepare
	q := `
	update call_channels set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetData. err: %v", err)
	}

	_, err = h.db.Exec(q, tmpData, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetData. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetDataItem sets the item into the channel's data
func (h *handler) ChannelSetDataItem(ctx context.Context, id string, key string, value interface{}) error {
	//prepare
	q := fmt.Sprintf("update call_channels set data = json_set(data, '$.%s', ?), tm_update = ? where id = ?", key)

	_, err := h.db.Exec(q, value, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDataItem. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStasis sets the stasis
func (h *handler) ChannelSetStasis(ctx context.Context, id, stasis string) error {
	//prepare
	q := `
	update call_channels set
		stasis_name = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, stasis, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasis. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStateAnswer sets the given channel's state and tm_answer
func (h *handler) ChannelSetStateAnswer(ctx context.Context, id string, state ari.ChannelState) error {

	q := `
	update call_channels set
		state = ?,
		tm_update = ?,
		tm_answer = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, state, ts, ts, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStateUp. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStateRinging sets the given channel's state and tm_ringing
func (h *handler) ChannelSetStateRinging(ctx context.Context, id string, state ari.ChannelState) error {

	q := `
	update call_channels set
		state = ?,
		tm_update = ?,
		tm_ringing = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, state, ts, ts, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStateRinging. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetBridgeID sets the bridge_id
func (h *handler) ChannelSetBridgeID(ctx context.Context, id, bridgeID string) error {
	//prepare
	q := `
	update call_channels set
		bridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, bridgeID, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetBridgeID. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetDirection sets the channel's direction
func (h *handler) ChannelSetDirection(ctx context.Context, id string, direction channel.Direction) error {
	//prepare
	q := `
	update call_channels set
		direction = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, direction, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDirection. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetType sets the bridge_id
func (h *handler) ChannelSetType(ctx context.Context, id string, cType channel.Type) error {
	//prepare
	q := `
	update call_channels set
		type = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, string(cType), h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetType. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetSIPTransport sets the channel's sip_transport
func (h *handler) ChannelSetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error {
	//prepare
	q := `
	update call_channels set
		sip_transport = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, transport, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetSIPTransport. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetSIPCallID sets the channel's sip_call_id
func (h *handler) ChannelSetSIPCallID(ctx context.Context, id string, sipID string) error {
	//prepare
	q := `
	update call_channels set
		sip_call_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, sipID, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetSIPCallID. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetPlaybackID sets the channel's playback_id
func (h *handler) ChannelSetPlaybackID(ctx context.Context, id string, playbackID string) error {
	//prepare
	q := `
	update call_channels set
		playback_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, playbackID, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetPlaybackID. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelEnd updates the channel end.
func (h *handler) ChannelEndAndDelete(ctx context.Context, id string, hangup ari.ChannelCause) error {
	// prepare
	q := `
	update call_channels set
		hangup_cause = ?,
		tm_update = ?,
		tm_end = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, hangup, ts, ts, ts, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelEnd. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStasisInfoAndSIPInfo sets stasis info and SIP info
func (h *handler) ChannelSetStasisInfo(ctx context.Context, id string, chType channel.Type, stasisName string, stasisData map[channel.StasisDataType]string, direction channel.Direction) error {
	//prepare
	q := `
	update call_channels set
		type = ?,

		stasis_name = ?,
		stasis_data = ?,

		direction = ?,

		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(stasisData)
	if err != nil {
		return fmt.Errorf("ChannelSetStasisInfo: Could not marshal the stasis_data. err: %v", err)
	}

	_, err = h.db.Exec(q,
		chType,

		stasisName,
		tmpData,

		direction,

		h.utilHandler.TimeGetCurTime(),
		id,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasisInfo. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// channelGetFromCache returns channel from the cache if possible.
func (h *handler) channelGetFromCache(ctx context.Context, id string) (*channel.Channel, error) {

	// get from cache
	res, err := h.cache.ChannelGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// channelGetFromDB returns channel from the DB.
func (h *handler) channelGetFromDB(ctx context.Context, id string) (*channel.Channel, error) {

	row, err := h.db.Query(channelSelect, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChannelGet. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.channelGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// channelUpdateToCache gets the channel from the DB and update the cache.
func (h *handler) channelUpdateToCache(ctx context.Context, id string) error {

	res, err := h.channelGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.channelSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// channelSetToCache sets the given channel to the cache
func (h *handler) channelSetToCache(ctx context.Context, channel *channel.Channel) error {
	if err := h.cache.ChannelSet(ctx, channel); err != nil {
		return err
	}

	return nil
}

// ChannelSetMuteDirection sets the channel's mute_direction
func (h *handler) ChannelSetMuteDirection(ctx context.Context, id string, muteDirection channel.MuteDirection) error {
	//prepare
	q := `
	update call_channels set
		mute_direction = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, muteDirection, h.utilHandler.TimeGetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetMuteDirection. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}
