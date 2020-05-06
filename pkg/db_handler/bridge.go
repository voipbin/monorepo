package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
)

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) BridgeCreate(ctx context.Context, b *bridge.Bridge) error {
	q := `insert into cm_bridges(
		asterisk_id,
		id,
		name,

		type,
		tech,
		class,
		creator,

		video_mode,
		video_source_id,

		channels,

		tm_create
	) values(
		?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?,
		?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. BridgeCreate. err: %v", err)
	}
	defer stmt.Close()

	tmpChannels, err := json.Marshal(b.Channels)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. BridgeCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		b.AsteriskID,
		b.ID,
		b.Name,

		b.Type,
		b.Tech,
		b.Class,
		b.Creator,

		b.VideoMode,
		b.VideoSourceID,

		tmpChannels,

		b.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not execute query. BridgeCreate. err: %v", err)
	}

	return nil
}

// BridgeGet returns bridge.
func (h *handler) BridgeGet(ctx context.Context, asteriskID, id string) (*bridge.Bridge, error) {

	// prepare
	q := `
	select
		asterisk_id,
		id,
		name,

		type,
		tech,
		class,
		creator,

		video_mode,
		video_source_id,

		channels,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		cm_bridges
	where
	asterisk_id = ? and id = ?`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not prepare. BridgeGet. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, asteriskID, id)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not query. BridgeGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	var channels string
	res := &bridge.Bridge{}
	if err := row.Scan(
		&res.AsteriskID,
		&res.ID,
		&res.Name,

		&res.Type,
		&res.Tech,
		&res.Class,
		&res.Creator,

		&res.VideoMode,
		&res.VideoSourceID,

		&channels,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. BridgeGet. err: %v", err)
	}

	if err := json.Unmarshal([]byte(channels), &res.Channels); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not unmarshal the result. BridgeGet. err: %v", err)
	}
	if res.Channels == nil {
		res.Channels = []string{}
	}

	return res, nil
}

// BridgeEnd updates the bridge end.
func (h *handler) BridgeEnd(ctx context.Context, asteriskID, id, timestamp string) error {
	// prepare
	q := `
	update cm_bridges set
		tm_update = ?,
		tm_delete = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. BridgeEnd. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, getCurTime(), timestamp, asteriskID, id)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not query. BridgeEnd. err: %v", err)
	}

	return nil
}
