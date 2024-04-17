package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdialtarget"
)

const (
	// select query for outdial get
	outdialTargetSelect = `
	select
		id,
		outdial_id,

		name,
		detail,

		data,
		status,

		destination_0,
		destination_1,
		destination_2,
		destination_3,
		destination_4,

		try_count_0,
		try_count_1,
		try_count_2,
		try_count_3,
		try_count_4,

		tm_create,
		tm_update,
		tm_delete
	from
		outdialtargets
	`

	// select query for available outdialtarget get
	outdialTargetSelectAvailable = `
	select
		id,
		outdial_id,

		name,
		detail,

		data,
		status,

		destination_0,
		destination_1,
		destination_2,
		destination_3,
		destination_4,

		try_count_0,
		try_count_1,
		try_count_2,
		try_count_3,
		try_count_4,

		tm_create,
		tm_update,
		tm_delete,

		case when destination_0 is null then 0 when try_count_0 < ? then 1 else 0 end as des_0,
		case when destination_1 is null then 0 when try_count_1 < ? then 1 else 0 end as des_1,
		case when destination_2 is null then 0 when try_count_2 < ? then 1 else 0 end as des_2,
		case when destination_3 is null then 0 when try_count_3 < ? then 1 else 0 end as des_3,
		case when destination_4 is null then 0 when try_count_4 < ? then 1 else 0 end as des_4
	from
		outdialtargets
	having
		status = "idle"
		and des_0 + des_1 + des_2 + des_3 + des_4 > 0
		and outdial_id = ?
	order by tm_update asc
	limit ?
	`
)

// outdialTargetGetFromRow gets the outdialtarget from the row.
func (h *handler) outdialTargetGetFromRow(row *sql.Rows) (*outdialtarget.OutdialTarget, error) {
	var destination0 sql.NullString
	var destination1 sql.NullString
	var destination2 sql.NullString
	var destination3 sql.NullString
	var destination4 sql.NullString

	res := &outdialtarget.OutdialTarget{}
	if err := row.Scan(
		&res.ID,
		&res.OutdialID,

		&res.Name,
		&res.Detail,

		&res.Data,
		&res.Status,

		&destination0,
		&destination1,
		&destination2,
		&destination3,
		&destination4,

		&res.TryCount0,
		&res.TryCount1,
		&res.TryCount2,
		&res.TryCount3,
		&res.TryCount4,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialTargetGetFromRow. err: %v", err)
	}

	if destination0.Valid {
		if errDestination := json.Unmarshal([]byte(destination0.String), &res.Destination0); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination0. outdialTargetGetFromRow. err: %v", errDestination)
		}
	}

	if destination1.Valid {
		if errDestination := json.Unmarshal([]byte(destination1.String), &res.Destination1); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination1. outdialTargetGetFromRow. err: %v", errDestination)
		}
	}

	if destination2.Valid {
		if errDestination := json.Unmarshal([]byte(destination2.String), &res.Destination2); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination2. outdialTargetGetFromRow. err: %v", errDestination)
		}
	}

	if destination3.Valid {
		if errDestination := json.Unmarshal([]byte(destination3.String), &res.Destination3); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination3. outdialTargetGetFromRow. err: %v", errDestination)
		}
	}

	if destination4.Valid {
		if errDestination := json.Unmarshal([]byte(destination4.String), &res.Destination4); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination4. outdialTargetGetFromRow. err: %v", errDestination)
		}
	}

	return res, nil
}

// outdialTargetGetFromRow gets the outdialtarget from the row.
func (h *handler) outdialTargetGetFromRowForAvailable(row *sql.Rows) (*outdialtarget.OutdialTarget, error) {
	var destination0 sql.NullString
	var destination1 sql.NullString
	var destination2 sql.NullString
	var destination3 sql.NullString
	var destination4 sql.NullString

	var des0 int
	var des1 int
	var des2 int
	var des3 int
	var des4 int

	res := &outdialtarget.OutdialTarget{}
	if err := row.Scan(
		&res.ID,
		&res.OutdialID,

		&res.Name,
		&res.Detail,

		&res.Data,
		&res.Status,

		&destination0,
		&destination1,
		&destination2,
		&destination3,
		&destination4,

		&res.TryCount0,
		&res.TryCount1,
		&res.TryCount2,
		&res.TryCount3,
		&res.TryCount4,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,

		&des0,
		&des1,
		&des2,
		&des3,
		&des4,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialTargetGetFromRowForAvailable. err: %v", err)
	}

	if destination0.Valid {
		if errDestination := json.Unmarshal([]byte(destination0.String), &res.Destination0); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination0. outdialTargetGetFromRowForAvailable. err: %v", errDestination)
		}
	}

	if destination1.Valid {
		if errDestination := json.Unmarshal([]byte(destination1.String), &res.Destination1); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination1. outdialTargetGetFromRowForAvailable. err: %v", errDestination)
		}
	}

	if destination2.Valid {
		if errDestination := json.Unmarshal([]byte(destination2.String), &res.Destination2); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination2. outdialTargetGetFromRowForAvailable. err: %v", errDestination)
		}
	}

	if destination3.Valid {
		if errDestination := json.Unmarshal([]byte(destination3.String), &res.Destination3); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination3. outdialTargetGetFromRowForAvailable. err: %v", errDestination)
		}
	}

	if destination4.Valid {
		if errDestination := json.Unmarshal([]byte(destination4.String), &res.Destination4); errDestination != nil {
			return nil, fmt.Errorf("could not unmarshal the destination4. outdialTargetGetFromRowForAvailable. err: %v", errDestination)
		}
	}

	return res, nil
}

// OutdialTargetCreate insert a new outdialtarget record
func (h *handler) OutdialTargetCreate(ctx context.Context, t *outdialtarget.OutdialTarget) error {

	q := `insert into outdialtargets(
		id,
		outdial_id,

		name,
		detail,

		data,
		status,

		destination_0,
		destination_1,
		destination_2,
		destination_3,
		destination_4,

		try_count_0,
		try_count_1,
		try_count_2,
		try_count_3,
		try_count_4,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. OutdialTargetCreate. err: %v", err)
	}
	defer stmt.Close()

	var destination0 []byte = nil
	if t.Destination0 != nil {
		destination0, err = json.Marshal(t.Destination0)
		if err != nil {
			return fmt.Errorf("could not marshal destination0. OutdialTargetCreate. err: %v", err)
		}
	}

	var destination1 []byte = nil
	if t.Destination1 != nil {
		destination1, err = json.Marshal(t.Destination1)
		if err != nil {
			return fmt.Errorf("could not marshal destination1. OutdialTargetCreate. err: %v", err)
		}
	}

	var destination2 []byte = nil
	if t.Destination2 != nil {
		destination2, err = json.Marshal(t.Destination2)
		if err != nil {
			return fmt.Errorf("could not marshal destination2. OutdialTargetCreate. err: %v", err)
		}
	}

	var destination3 []byte = nil
	if t.Destination3 != nil {
		destination3, err = json.Marshal(t.Destination3)
		if err != nil {
			return fmt.Errorf("could not marshal destination3. OutdialTargetCreate. err: %v", err)
		}
	}

	var destination4 []byte = nil
	if t.Destination4 != nil {
		destination4, err = json.Marshal(t.Destination4)
		if err != nil {
			return fmt.Errorf("could not marshal destination4. OutdialTargetCreate. err: %v", err)
		}
	}

	_, err = stmt.ExecContext(ctx,
		t.ID.Bytes(),
		t.OutdialID.Bytes(),

		t.Name,
		t.Detail,

		t.Data,
		t.Status,

		destination0,
		destination1,
		destination2,
		destination3,
		destination4,

		t.TryCount0,
		t.TryCount1,
		t.TryCount2,
		t.TryCount3,
		t.TryCount4,

		t.TMCreate,
		t.TMUpdate,
		t.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. OutdialTargetCreate. err: %v", err)
	}

	_ = h.outdialTargetUpdateToCache(ctx, t.ID)

	return nil
}

// outdialTargetUpdateToCache gets the outdial from the DB and update the cache.
func (h *handler) outdialTargetUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.outdialTargetGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.outdialTargetSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// outdialTargetSetToCache sets the given outdialTarget to the cache
func (h *handler) outdialTargetSetToCache(ctx context.Context, t *outdialtarget.OutdialTarget) error {
	if err := h.cache.OutdialTargetSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// outdialTargetGetFromCache returns outdialTarget from the cache if possible.
func (h *handler) outdialTargetGetFromCache(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error) {

	// get from cache
	res, err := h.cache.OutdialTargetGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetGetFromDB gets the outdialTarget info from the db.
func (h *handler) outdialTargetGetFromDB(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", outdialTargetSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. outdialTargetGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. outdialTargetGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OutdialTargetGet returns outdialtarget.
func (h *handler) OutdialTargetGet(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error) {

	res, err := h.outdialTargetGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.outdialTargetGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetUpdateToCache(ctx, id)

	return res, nil
}

// OutdialTargetGetsByOutdialID returns list of outdialtargets.
func (h *handler) OutdialTargetGetsByOutdialID(ctx context.Context, outdialID uuid.UUID, token string, limit uint64) ([]*outdialtarget.OutdialTarget, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and outdial_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, outdialTargetSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, outdialID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetGetsByOutdialID. err: %v", err)
	}
	defer rows.Close()

	var res []*outdialtarget.OutdialTarget
	for rows.Next() {
		u, err := h.outdialTargetGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OutdialTargetGetsByOutdialID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// OutdialTargetUpdateDestinations updates outdialtarget's destinations.
func (h *handler) OutdialTargetUpdateDestinations(
	ctx context.Context,
	id uuid.UUID,
	destination0 *commonaddress.Address,
	destination1 *commonaddress.Address,
	destination2 *commonaddress.Address,
	destination3 *commonaddress.Address,
	destination4 *commonaddress.Address,
) error {
	q := `
	update outdialtargets set
		destination_0 = ?,
		destination_1 = ?,
		destination_2 = ?,
		destination_3 = ?,
		destination_4 = ?,
		tm_update = ?
	where
		id = ?
	`

	var err error
	var tmpDestination0 []byte = nil
	if destination0 != nil {
		tmpDestination0, err = json.Marshal(destination0)
		if err != nil {
			return fmt.Errorf("could not marshal destination0. OutdialTargetCreate. err: %v", err)
		}
	}

	var tmpDestination1 []byte = nil
	if destination1 != nil {
		tmpDestination1, err = json.Marshal(destination1)
		if err != nil {
			return fmt.Errorf("could not marshal destination1. OutdialTargetCreate. err: %v", err)
		}
	}

	var tmpDestination2 []byte = nil
	if destination2 != nil {
		tmpDestination2, err = json.Marshal(destination2)
		if err != nil {
			return fmt.Errorf("could not marshal destination2. OutdialTargetCreate. err: %v", err)
		}
	}

	var tmpDestination3 []byte = nil
	if destination3 != nil {
		tmpDestination3, err = json.Marshal(destination3)
		if err != nil {
			return fmt.Errorf("could not marshal destination3. OutdialTargetCreate. err: %v", err)
		}
	}

	var tmpDestination4 []byte = nil
	if destination4 != nil {
		tmpDestination4, err = json.Marshal(destination4)
		if err != nil {
			return fmt.Errorf("could not marshal destination4. OutdialTargetCreate. err: %v", err)
		}
	}

	if _, err := h.db.Exec(
		q,
		tmpDestination0,
		tmpDestination1,
		tmpDestination2,
		tmpDestination3,
		tmpDestination4,
		GetCurTime(),
		id.Bytes(),
	); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetUpdateDestinations. err: %v", err)
	}

	// set to the cache
	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil

}

// OutdialTargetUpdateStatus updates outdialtarget's status.
func (h *handler) OutdialTargetUpdateStatus(ctx context.Context, id uuid.UUID, status outdialtarget.Status) error {
	q := `
	update outdialtargets set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, status, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetUpdateDestinations. err: %v", err)
	}

	// set to the cache
	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil
}

// OutdialTargetUpdateBasicInfo updates outdialtarget's basic info.
func (h *handler) OutdialTargetUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update outdialtargets set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil
}

// OutdialTargetUpdateData updates outdialtarget's data.
func (h *handler) OutdialTargetUpdateData(ctx context.Context, id uuid.UUID, data string) error {
	q := `
	update outdialtargets set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, data, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetUpdateData. err: %v", err)
	}

	// set to the cache
	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil
}

// OutdialTargetDelete delets the outdialtarget.
func (h *handler) OutdialTargetDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update outdialtargets set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := GetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetDelete. err: %v", err)
	}

	// set to the cache
	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil
}

// OutdialTargetUpdateProgressing updates outdialtarget's basic info.
func (h *handler) OutdialTargetUpdateProgressing(ctx context.Context, id uuid.UUID, destinationIndex int) error {
	q := fmt.Sprintf(`
	update outdialtargets set
		try_count_%d = try_count_%d + 1,
		status = ?,
		tm_update = ?
	where
		id = ?
	`, destinationIndex, destinationIndex)

	if _, err := h.db.Exec(q, outdialtarget.StatusProgressing, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil
}

// OutdialTargetGetAvailable returns available outdialtargets.
func (h *handler) OutdialTargetGetAvailable(
	ctx context.Context,
	outdialID uuid.UUID,
	tryCount0 int,
	tryCount1 int,
	tryCount2 int,
	tryCount3 int,
	tryCount4 int,
	limit uint64,
) ([]*outdialtarget.OutdialTarget, error) {

	// prepare
	stmt, err := h.db.PrepareContext(ctx, outdialTargetSelectAvailable)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. OutdialTargetGetAvailable. err: %v", err)
	}
	defer stmt.Close()

	// query
	rows, err := stmt.QueryContext(ctx, tryCount0, tryCount1, tryCount2, tryCount3, tryCount4, outdialID.Bytes(), limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetGetAvailable. err: %v", err)
	}
	defer rows.Close()

	res := []*outdialtarget.OutdialTarget{}
	for rows.Next() {
		u, err := h.outdialTargetGetFromRowForAvailable(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OutdialTargetGetAvailable. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
