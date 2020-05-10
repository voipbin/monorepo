package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) ChannelCreate(ctx context.Context, channel *channel.Channel) error {
	q := `insert into cm_channels(
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
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	tmpData, err := json.Marshal(channel.Data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
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

	return nil
}

// ChannelGet returns channel.
func (h *handler) ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error) {

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
		cm_channels
	where
	asterisk_id = ? and id = ?`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, asteriskID, id)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not query. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	return h.channelGetFromRow(row)
}

// ChannelGet returns channel.
func (h *handler) ChannelGetByID(ctx context.Context, id string) (*channel.Channel, error) {

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
		cm_channels
	where
		id = ?`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not query. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	return h.channelGetFromRow(row)
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
func (h *handler) ChannelIsExist(id, asteriskID string, timeout time.Duration) bool {
	// check the channel is exists
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := h.ChannelGetUntilTimeout(ctx, asteriskID, id)
	if err != nil {
		return false
	}
	return true
}

// ChannelGetUntilTimeoutWithStasis gets the stasis channel until the ctx is timed out.
func (h *handler) ChannelGetUntilTimeoutWithStasis(ctx context.Context, asteriskID, id string) (*channel.Channel, error) {

	chanChannel := make(chan *channel.Channel)

	go func() {
		for {
			channel, err := h.ChannelGet(ctx, asteriskID, id)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if channel.Stasis == "" {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			chanChannel <- channel
		}
	}()

	select {
	case res := <-chanChannel:
		return res, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("could not get channel. err: tiemout")
	}
}

// ChannelGetUntilTimeout gets the channel until the ctx is timed out.
func (h *handler) ChannelGetUntilTimeout(ctx context.Context, asteriskID, id string) (*channel.Channel, error) {

	chanChannel := make(chan *channel.Channel)

	go func() {
		for {
			channel, err := h.ChannelGet(ctx, asteriskID, id)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			chanChannel <- channel
		}
	}()

	select {
	case res := <-chanChannel:
		return res, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("could not get channel. err: tiemout")
	}
}

// ChannelSetData sets the data
func (h *handler) ChannelSetData(ctx context.Context, asteriskID, id string, data map[string]interface{}) error {
	//prepare
	q := `
	update cm_channels set
		data = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetData. err: %v", err)
	}
	defer stmt.Close()

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetData. err: %v", err)
	}

	// execute
	_, err = stmt.ExecContext(ctx, tmpData, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetData. err: %v", err)
	}

	return nil
}

// ChannelSetStasis sets the stasis
func (h *handler) ChannelSetStasis(ctx context.Context, asteriskID, id, stasis string) error {
	//prepare
	q := `
	update cm_channels set
		stasis = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetStasis. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, stasis, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasis. err: %v", err)
	}

	return nil
}

func (h *handler) ChannelSetState(ctx context.Context, asteriskID, id, timestamp string, state ari.ChannelState) error {

	var q string
	switch state {
	case ari.ChannelStateUp:
		q = `
		update cm_channels set
			state = ?,
			tm_update = ?,
			tm_answer = ?
		where
			asterisk_id = ?
			and id = ?
		`
	case ari.ChannelStateRing, ari.ChannelStateRinging:
		q = `
		update cm_channels set
			state = ?,
			tm_update = ?,
			tm_ringing = ?
		where
			asterisk_id = ?
			and id = ?
		`
	default:
		return fmt.Errorf("no match state. ChannelSetState. state: %s", state)
	}

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetState. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, string(state), timestamp, timestamp, asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetState. err: %v", err)
	}

	return nil
}

// ChannelSetBridgeID sets the bridge_id
func (h *handler) ChannelSetBridgeID(ctx context.Context, asteriskID, id, bridgeID string) error {
	//prepare
	q := `
	update cm_channels set
		bridge_id = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetBridgeID. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, bridgeID, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetBridgeID. err: %v", err)
	}

	return nil
}

// ChannelEnd updates the channel end.
func (h *handler) ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup ari.ChannelCause) error {
	// prepare
	q := `
	update cm_channels set
		hangup_cause = ?,
		tm_update = ?,
		tm_end = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, hangup, timestamp, timestamp, asteriskID, id)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not query. err: %v", err)
	}

	return nil
}

// ChannelSetDataAndStasis sets the data and stasis
func (h *handler) ChannelSetDataAndStasis(ctx context.Context, asteriskID, id string, data map[string]interface{}, stasis string) error {
	//prepare
	q := `
	update cm_channels set
		data = ?,
		stasis = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetDataAndStasis. err: %v", err)
	}
	defer stmt.Close()

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetDataAndStasis. err: %v", err)
	}

	// execute
	_, err = stmt.ExecContext(ctx, tmpData, stasis, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDataAndStasis. err: %v", err)
	}

	return nil
}
