package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// ConferenceCreate creates a new conference record.
func (h *handler) ConferenceCreate(ctx context.Context, cf *conference.Conference) error {
	q := `insert into cm_conferences(
		id,
		type,

		status,
		name,
		detail,
		data,

		bridge_ids,
		call_ids,

		tm_create
	) values(
		?, ?,
		?, ?, ?, ?,
		?, ?,
		?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ConferenceCreate. err: %v", err)
	}
	defer stmt.Close()

	data, err := json.Marshal(cf.Data)
	if err != nil {
		return fmt.Errorf("could not marshal data. ConferenceCreate. err: %v", err)
	}

	bridgeIDs, err := json.Marshal(cf.BridgeIDs)
	if err != nil {
		return fmt.Errorf("could not marshal bridges. ConferenceCreate. err: %v", err)
	}

	callIDs, err := json.Marshal(cf.CallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ConferenceCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		cf.ID.Bytes(),
		cf.Type,

		cf.Status,
		cf.Name,
		cf.Detail,
		data,

		bridgeIDs,
		callIDs,

		cf.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. ConferenceCreate. err: %v", err)
	}

	return nil
}

// ConferenceGet gets conference.
func (h *handler) ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {

	// prepare
	q := `
	select
		id,
		type,

		status,
		name,
		detail,
		data,

		bridge_ids,
		call_ids,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete

	from
		cm_conferences
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. ConferenceGet. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferenceGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.conferenceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConferenceGet, err: %v", err)
	}

	return res, nil
}

// conferenceGetFromRow gets the call from the row.
func (h *handler) conferenceGetFromRow(row *sql.Rows) (*conference.Conference, error) {
	var data string
	var bridges string
	var calls string
	res := &conference.Conference{}
	if err := row.Scan(
		&res.ID,
		&res.Type,

		&res.Status,
		&res.Name,
		&res.Detail,
		&data,

		&bridges,
		&calls,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. conferenceGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(bridges), &res.BridgeIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. conferenceGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(calls), &res.CallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. conferenceGetFromRow. err: %v", err)
	}

	return res, nil
}

// ConferenceAddCallID adds the call id to the conference.
func (h *handler) ConferenceAddCallID(ctx context.Context, id, callID uuid.UUID) error {
	// prepare
	q := `
	update cm_conferences set
		call_ids = json_array_append(
			call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. ConferenceAddCallID. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, callID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("dbhandler: Could not query. ConferenceAddCallID. err: %v", err)
	}

	return nil
}

// ConferenceRemoveCallID removes the call id from the conference.
func (h *handler) ConferenceRemoveCallID(ctx context.Context, id, callID uuid.UUID) error {
	// prepare
	q := `
	update cm_conferences set
		call_ids = json_remove(
			call_ids, replace(
				json_search(
					call_ids,
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
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. ConferenceRemoveCallID. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, callID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("dbhandler: Could not query. ConferenceRemoveCallID. err: %v", err)
	}

	return nil
}

// ConferenceSetStatus sets status
func (h *handler) ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error {
	//prepare
	q := `
	update cm_conferences set
		status = ?,
		tm_update = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ConferenceSetStatus. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, status, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetStatus. err: %v", err)
	}

	return nil
}

// ConferenceEnd ends the conference
func (h *handler) ConferenceEnd(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update cm_conferences set
		status = ?,
		tm_delete = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ConferenceEnd. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, conference.StatusTerminated, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceEnd. err: %v", err)
	}

	return nil
}
