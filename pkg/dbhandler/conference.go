package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// ConferenceGetsByUserID returns list of conferences.
func (h *handler) ConferenceGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Conference, error) {

	// prepare
	q := `
	select
		id,
		user_id,
		type,

		status,
		name,
		detail,

		call_ids,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		conferences
	where
		user_id = ? and tm_create < ?
	order by
		tm_create desc, id desc
	limit ?

	`

	rows, err := h.db.Query(q, userID, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferenceGetsByUserID. err: %v", err)
	}
	defer rows.Close()

	var res []*models.Conference
	for rows.Next() {
		u, err := h.conferenceGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ConferenceGetsByUserID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// conferenceGetFromRow gets the conference from the row.
func (h *handler) conferenceGetFromRow(row *sql.Rows) (*models.Conference, error) {
	var calls string
	res := &models.Conference{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.Type,

		&res.Status,
		&res.Name,
		&res.Detail,

		&calls,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(calls), &res.CallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the call_ids. conferenceGetFromRow. err: %v", err)
	}

	return res, nil
}
