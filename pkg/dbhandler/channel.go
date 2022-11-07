package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
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

		tm_create,
		tm_update,

		tm_answer,
		tm_ringing,
		tm_end

	from
		channels
	where
		id = ?
	`
)

// channelGetFromRow gets the channel from the row.
func (h *handler) channelGetFromRow(row *sql.Rows) (*channel.Channel, error) {
	var data sql.NullString
	var stasisData sql.NullString

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

		&res.TMCreate,
		&res.TMUpdate,

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
		res.StasisData = map[string]string{}
	}

	return res, nil
}

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) ChannelCreate(ctx context.Context, c *channel.Channel) error {
	q := `insert into channels(
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

		tm_create,
		tm_update,

		tm_answer,
		tm_ringing,
		tm_end

	) values(
		?, ?, ?, ?, ?,
		?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?, ?,
		?, ?,
		?,
		?, ?,
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

		c.TMCreate,
		c.TMUpdate,

		c.TMAnswer,
		c.TMRinging,
		c.TMEnd,
	)
	if err != nil {
		return fmt.Errorf("ChannelCreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, c.ID)

	return nil
}

// ChannelGet returns channel.
func (h *handler) ChannelGet(ctx context.Context, id string) (*channel.Channel, error) {

	res, err := h.ChannelGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.ChannelGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.ChannelSetToCache(ctx, res)

	return res, nil
}

// ChannelIsExist returns true if the channel exist within timeout.
func (h *handler) ChannelIsExist(id string, timeout time.Duration) bool {
	// check the channel is exists
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := h.ChannelGetUntilTimeout(ctx, id)

	return err == nil
}

// ChannelGetUntilTimeoutWithStasis gets the stasis channel until the ctx is timed out.
func (h *handler) ChannelGetUntilTimeoutWithStasis(ctx context.Context, id string) (*channel.Channel, error) {

	chanRes := make(chan *channel.Channel)
	chanStop := make(chan bool)

	go func() {
		for {
			select {
			case <-chanStop:
				return

			default:
				tmp, err := h.ChannelGet(ctx, id)
				if err != nil || tmp.StasisName == "" {
					time.Sleep(defaultDelayTimeout)
					continue
				}

				chanRes <- tmp
				return
			}
		}
	}()

	select {
	case res := <-chanRes:
		return res, nil

	case <-ctx.Done():
		chanStop <- true
		return nil, fmt.Errorf("could not get channel. err: tiemout")
	}
}

// ChannelGetUntilTimeout gets the channel until the ctx is timed out.
func (h *handler) ChannelGetUntilTimeout(ctx context.Context, id string) (*channel.Channel, error) {

	chanRes := make(chan *channel.Channel)
	chanStop := make(chan bool)

	go func() {
		for {
			select {
			case <-chanStop:
				return

			default:
				tmp, err := h.ChannelGet(ctx, id)
				if err != nil {
					time.Sleep(defaultDelayTimeout)
					continue
				}

				chanRes <- tmp
				return
			}
		}
	}()

	select {
	case res := <-chanRes:
		return res, nil
	case <-ctx.Done():
		chanStop <- true
		return nil, fmt.Errorf("could not get channel. err: tiemout")
	}
}

// ChannelSetData sets the data
func (h *handler) ChannelSetData(ctx context.Context, id string, data map[string]interface{}) error {
	//prepare
	q := `
	update channels set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetData. err: %v", err)
	}

	_, err = h.db.Exec(q, tmpData, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetData. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetDataItem sets the item into the channel's data
func (h *handler) ChannelSetDataItem(ctx context.Context, id string, key string, value interface{}) error {
	//prepare
	q := fmt.Sprintf("update channels set data = json_set(data, '$.%s', ?), tm_update = ? where id = ?", key)

	_, err := h.db.Exec(q, value, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDataItem. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStasis sets the stasis
func (h *handler) ChannelSetStasis(ctx context.Context, id, stasis string) error {
	//prepare
	q := `
	update channels set
		stasis_name = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, stasis, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasis. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

func (h *handler) ChannelSetState(ctx context.Context, id, timestamp string, state ari.ChannelState) error {

	var q string
	switch state {
	case ari.ChannelStateUp:
		q = `
		update channels set
			state = ?,
			tm_update = ?,
			tm_answer = ?
		where
			id = ?
		`
	case ari.ChannelStateRing, ari.ChannelStateRinging:
		q = `
		update channels set
			state = ?,
			tm_update = ?,
			tm_ringing = ?
		where
			id = ?
		`
	default:
		return fmt.Errorf("no match state. ChannelSetState. state: %s", state)
	}

	_, err := h.db.Exec(q, string(state), timestamp, timestamp, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetState. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetBridgeID sets the bridge_id
func (h *handler) ChannelSetBridgeID(ctx context.Context, id, bridgeID string) error {
	//prepare
	q := `
	update channels set
		bridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, bridgeID, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetBridgeID. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetDirection sets the channel's direction
func (h *handler) ChannelSetDirection(ctx context.Context, id string, direction channel.Direction) error {
	//prepare
	q := `
	update channels set
		direction = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, direction, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDirection. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetType sets the bridge_id
func (h *handler) ChannelSetType(ctx context.Context, id string, cType channel.Type) error {
	//prepare
	q := `
	update channels set
		type = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, string(cType), h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetType. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetSIPTransport sets the channel's sip_transport
func (h *handler) ChannelSetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error {
	//prepare
	q := `
	update channels set
		sip_transport = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, transport, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetSIPTransport. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetSIPCallID sets the channel's sip_call_id
func (h *handler) ChannelSetSIPCallID(ctx context.Context, id string, sipID string) error {
	//prepare
	q := `
	update channels set
		sip_call_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, sipID, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetSIPCallID. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetPlaybackID sets the channel's playback_id
func (h *handler) ChannelSetPlaybackID(ctx context.Context, id string, playbackID string) error {
	//prepare
	q := `
	update channels set
		playback_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, playbackID, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetPlaybackID. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelEnd updates the channel end.
func (h *handler) ChannelEnd(ctx context.Context, id, timestamp string, hangup ari.ChannelCause) error {
	// prepare
	q := `
	update channels set
		hangup_cause = ?,
		tm_update = ?,
		tm_end = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, hangup, timestamp, timestamp, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelEnd. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStasisNameAndStasisData sets the data and stasis
func (h *handler) ChannelSetStasisNameAndStasisData(ctx context.Context, id string, stasisName string, stasisData map[string]string) error {
	//prepare
	q := `
	update channels set
		stasis_name = ?,
		stasis_data = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(stasisData)
	if err != nil {
		return fmt.Errorf("ChannelSetStasisNameAndStasisData: Could not marshal the stasis_data. err: %v", err)
	}

	_, err = h.db.Exec(q, stasisName, tmpData, h.util.GetCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasisNameAndStasisData. err: %v", err)
	}

	// update the cache
	_ = h.ChannelUpdateToCache(ctx, id)

	return nil
}

// ChannelGetFromCache returns channel from the cache if possible.
func (h *handler) ChannelGetFromCache(ctx context.Context, id string) (*channel.Channel, error) {

	// get from cache
	res, err := h.cache.ChannelGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChannelGetFromDB returns channel from the DB.
func (h *handler) ChannelGetFromDB(ctx context.Context, id string) (*channel.Channel, error) {

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

// ChannelUpdateToCache gets the channel from the DB and update the cache.
func (h *handler) ChannelUpdateToCache(ctx context.Context, id string) error {

	res, err := h.ChannelGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.ChannelSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// ChannelSetToCache sets the given channel to the cache
func (h *handler) ChannelSetToCache(ctx context.Context, channel *channel.Channel) error {
	if err := h.cache.ChannelSet(ctx, channel); err != nil {
		return err
	}

	return nil
}
