package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
)

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) BridgeCreate(ctx context.Context, b *bridge.Bridge) error {
	q := `insert into cm_bridges(
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

		conference_id,
		conference_type,
		conference_join,

		tm_create
	) values(
		?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?,
		?, ?, ?,
		?
		)
	`

	tmpChannelIDs, err := json.Marshal(b.ChannelIDs)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. BridgeCreate. err: %v", err)
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

		b.ConferenceID.Bytes(),
		b.ConferenceType,
		b.ConferenceJoin,

		b.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeCreate. err: %v", err)
	}

	return nil
}

// BridgeGet returns bridge.
func (h *handler) BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error) {

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

		conference_id,
		conference_type,
		conference_join,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		cm_bridges
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

		&res.ConferenceID,
		&res.ConferenceType,
		&res.ConferenceJoin,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. BridgeGet. err: %v", err)
	}

	if err := json.Unmarshal([]byte(channelIDs), &res.ChannelIDs); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not unmarshal the result. BridgeGet. err: %v", err)
	}
	if res.ChannelIDs == nil {
		res.ChannelIDs = []string{}
	}

	return res, nil
}

// BridgeEnd updates the bridge end.
func (h *handler) BridgeEnd(ctx context.Context, id, timestamp string) error {
	// prepare
	q := `
	update cm_bridges set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, getCurTime(), timestamp, id)
	if err != nil {
		return fmt.Errorf("could not execute. BridgeEnd. err: %v", err)
	}

	return nil
}

// BridgeAddChannel adds the channel to the bridge.
func (h *handler) BridgeAddChannelID(ctx context.Context, id, channelID string) error {
	// prepare
	q := `
	update cm_bridges set
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

	return nil
}

// BridgeRemoveChannelID removes the channel from the bridge.
func (h *handler) BridgeRemoveChannelID(ctx context.Context, id, channelID string) error {
	// prepare
	q := `
	update cm_bridges set
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

	return nil
}

// BridgeGetUntilTimeout gets the bridge until the ctx is timed out.
func (h *handler) BridgeGetUntilTimeout(ctx context.Context, id string) (*bridge.Bridge, error) {

	chanBridge := make(chan *bridge.Bridge)

	go func() {
		for {
			res, err := h.BridgeGet(ctx, id)
			if err != nil {
				time.Sleep(defaultDelayTimeout)
				continue
			}

			chanBridge <- res
		}
	}()

	select {
	case res := <-chanBridge:
		return res, nil
	case <-ctx.Done():
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
