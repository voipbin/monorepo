package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// CallGet returns call.
func (h *handler) CallsGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Call, error) {

	// prepare
	q := `
	select
		id,
		user_id,
		flow_id,
		conference_id,
		type,

		source,
		destination,

		status,
		direction,
		hangup_by,
		hangup_reason,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,

		coalesce(tm_progressing, '') as tm_progressing,
		coalesce(tm_ringing, '') as tm_ringing,
		coalesce(tm_hangup, '') as tm_hangup
	from
		calls
	where
		user_id = ? and tm_create < ?
	order by
		tm_create desc, id desc
	limit ?

	`

	rows, err := h.db.Query(q, userID, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. CallsGetsByUserID. err: %v", err)
	}
	defer rows.Close()

	var res []*models.Call
	for rows.Next() {
		u, err := h.callGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. CallsGetsByUserID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// callGetFromRow gets the call from the row.
func (h *handler) callGetFromRow(row *sql.Rows) (*models.Call, error) {
	var source string
	var destination string
	res := &models.Call{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.FlowID,
		&res.ConfID,
		&res.Type,

		&source,
		&destination,

		&res.Status,
		&res.Direction,
		&res.HangupBy,
		&res.HangupReason,

		&res.TMCreate,
		&res.TMUpdate,

		&res.TMProgressing,
		&res.TMRinging,
		&res.TMHangup,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. callGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. callGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(destination), &res.Destination); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. callGetFromRow. err: %v", err)
	}

	return res, nil
}
