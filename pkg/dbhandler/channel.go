package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
)

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) ChannelCreate(ctx context.Context, channel *channel.Channel) error {
	q := `insert into channels(
		asterisk_id,
		id,
		name,
		tech,

		src_name,
		src_number,
		dst_name,
		dst_number,

		state,
		data,
		stasis,
		bridge_id,

		dial_result,
		hangup_cause,
		tm_create
	) values(
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?
		)
	`

	tmpData, err := json.Marshal(channel.Data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. err: %v", err)
	}

	_, err = h.db.Exec(q,
		channel.AsteriskID,
		channel.ID,
		channel.Name,
		channel.Tech,

		channel.SourceName,
		channel.SourceNumber,
		channel.DestinationName,
		channel.DestinationNumber,

		channel.State,
		tmpData,
		channel.Stasis,
		channel.BridgeID,

		channel.DialResult,
		channel.HangupCause,

		channel.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not execute query. err: %v", err)
	}

	// update the cache
	h.ChannelUpdateCache(ctx, channel.ID)

	return nil
}

// ChannelGet returns channel.
func (h *handler) ChannelGet(ctx context.Context, id string) (*channel.Channel, error) {

	res, err := h.ChannelGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.ChannelGetFromDB(ctx, id)
	if err == nil {
		return res, nil
	}

	return nil, err
}

// channelGetFromRow gets the channel from the row.
func (h *handler) channelGetFromRow(row *sql.Rows) (*channel.Channel, error) {
	var data string
	res := &channel.Channel{}
	if err := row.Scan(
		&res.AsteriskID,
		&res.ID,
		&res.Name,
		&res.Tech,

		&res.SourceName,
		&res.SourceNumber,
		&res.DestinationName,
		&res.DestinationNumber,

		&res.State,
		&data,
		&res.Stasis,
		&res.BridgeID,

		&res.DialResult,
		&res.HangupCause,

		&res.TMCreate,
		&res.TMUpdate,

		&res.TMAnswer,
		&res.TMRinging,
		&res.TMEnd,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not unmarshal the result. err: %v", err)
	}
	if res.Data == nil {
		res.Data = make(map[string]interface{})
	}

	return res, nil
}

// ChannelIsExist returns true if the channel exist within timeout.
func (h *handler) ChannelIsExist(id string, timeout time.Duration) bool {
	// check the channel is exists
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := h.ChannelGetUntilTimeout(ctx, id)
	if err != nil {
		return false
	}
	return true
}

// ChannelGetUntilTimeoutWithStasis gets the stasis channel until the ctx is timed out.
func (h *handler) ChannelGetUntilTimeoutWithStasis(ctx context.Context, id string) (*channel.Channel, error) {

	chanRes := make(chan *channel.Channel)
	stop := false

	go func() {
		for {
			if stop == true {
				return
			}

			tmp, err := h.ChannelGet(ctx, id)
			if err != nil {
				time.Sleep(defaultDelayTimeout)
				continue
			}

			if tmp.Stasis == "" {
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
		return nil, fmt.Errorf("could not get channel. err: tiemout")
	}
}

// ChannelGetUntilTimeout gets the channel until the ctx is timed out.
func (h *handler) ChannelGetUntilTimeout(ctx context.Context, id string) (*channel.Channel, error) {

	chanRes := make(chan *channel.Channel)
	stop := false

	go func() {
		for {
			if stop == true {
				return
			}

			tmp, err := h.ChannelGet(ctx, id)
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

	_, err = h.db.Exec(q, tmpData, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetData. err: %v", err)
	}

	// update the cache
	h.ChannelUpdateCache(ctx, id)

	return nil
}

// ChannelSetStasis sets the stasis
func (h *handler) ChannelSetStasis(ctx context.Context, id, stasis string) error {
	//prepare
	q := `
	update channels set
		stasis = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, stasis, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasis. err: %v", err)
	}

	// update the cache
	h.ChannelUpdateCache(ctx, id)

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
	h.ChannelUpdateCache(ctx, id)

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

	_, err := h.db.Exec(q, bridgeID, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetBridgeID. err: %v", err)
	}

	// update the cache
	h.ChannelUpdateCache(ctx, id)

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
	h.ChannelUpdateCache(ctx, id)

	return nil
}

// ChannelSetDataAndStasis sets the data and stasis
func (h *handler) ChannelSetDataAndStasis(ctx context.Context, id string, data map[string]interface{}, stasis string) error {
	//prepare
	q := `
	update channels set
		data = ?,
		stasis = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetDataAndStasis. err: %v", err)
	}

	_, err = h.db.Exec(q, tmpData, stasis, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDataAndStasis. err: %v", err)
	}

	// update the cache
	h.ChannelUpdateCache(ctx, id)

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

	// prepare
	q := `select
	asterisk_id,
  id,
  name,
  tech,

  src_name,
  src_number,
  dst_name,
  dst_number,

  state,
	data,
	stasis,
	bridge_id,

  dial_result,
  hangup_cause,

	coalesce(tm_create, '') as tm_create,
	coalesce(tm_update, '') as tm_update,

	coalesce(tm_answer, '') as tm_answer,
	coalesce(tm_ringing, '') as tm_ringing,
	coalesce(tm_end, '') as tm_end

	from
		channels
	where
		id = ?
	`

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChannelGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.channelGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChannelUpdateCache gets the channel from the DB and update the cache.
func (h *handler) ChannelUpdateCache(ctx context.Context, id string) error {

	res, err := h.ChannelGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.cache.ChannelSet(ctx, res); err != nil {
		return err
	}

	return nil
}
