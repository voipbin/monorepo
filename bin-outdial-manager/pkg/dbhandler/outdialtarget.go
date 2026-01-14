package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-outdial-manager/models/outdialtarget"
)

const (
	outdialtargetsTable = "outdial_outdialtargets"

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
		outdial_outdialtargets
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
	res := &outdialtarget.OutdialTarget{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialTargetGetFromRow. err: %v", err)
	}

	return res, nil
}

// outdialTargetGetFromRowForAvailable gets the outdialtarget from the row for available query.
// This method handles the extra columns (des_0 to des_4) in the available query.
func (h *handler) outdialTargetGetFromRowForAvailable(row *sql.Rows) (*outdialtarget.OutdialTarget, error) {
	res := &outdialtarget.OutdialTarget{}
	var des0, des1, des2, des3, des4 sql.NullInt64

	var id, outdialID sql.NullString
	var name, detail, data, status sql.NullString
	var destination0, destination1, destination2, destination3, destination4 sql.NullString
	var tryCount0, tryCount1, tryCount2, tryCount3, tryCount4 sql.NullInt64
	var tmCreate, tmUpdate, tmDelete sql.NullString

	if err := row.Scan(
		&id,
		&outdialID,
		&name,
		&detail,
		&data,
		&status,
		&destination0,
		&destination1,
		&destination2,
		&destination3,
		&destination4,
		&tryCount0,
		&tryCount1,
		&tryCount2,
		&tryCount3,
		&tryCount4,
		&tmCreate,
		&tmUpdate,
		&tmDelete,
		&des0,
		&des1,
		&des2,
		&des3,
		&des4,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialTargetGetFromRowForAvailable. err: %v", err)
	}

	// Convert scanned values to struct
	if id.Valid {
		res.ID, _ = uuid.FromBytes([]byte(id.String))
	}
	if outdialID.Valid {
		res.OutdialID, _ = uuid.FromBytes([]byte(outdialID.String))
	}
	if name.Valid {
		res.Name = name.String
	}
	if detail.Valid {
		res.Detail = detail.String
	}
	if data.Valid {
		res.Data = data.String
	}
	if status.Valid {
		res.Status = outdialtarget.Status(status.String)
	}
	if tryCount0.Valid {
		res.TryCount0 = int(tryCount0.Int64)
	}
	if tryCount1.Valid {
		res.TryCount1 = int(tryCount1.Int64)
	}
	if tryCount2.Valid {
		res.TryCount2 = int(tryCount2.Int64)
	}
	if tryCount3.Valid {
		res.TryCount3 = int(tryCount3.Int64)
	}
	if tryCount4.Valid {
		res.TryCount4 = int(tryCount4.Int64)
	}
	if tmCreate.Valid {
		res.TMCreate = tmCreate.String
	}
	if tmUpdate.Valid {
		res.TMUpdate = tmUpdate.String
	}
	if tmDelete.Valid {
		res.TMDelete = tmDelete.String
	}

	// Parse destinations (JSON fields)
	if destination0.Valid && len(destination0.String) > 0 {
		if err := parseDestination(destination0.String, &res.Destination0); err != nil {
			return nil, fmt.Errorf("could not parse destination0: %v", err)
		}
	}
	if destination1.Valid && len(destination1.String) > 0 {
		if err := parseDestination(destination1.String, &res.Destination1); err != nil {
			return nil, fmt.Errorf("could not parse destination1: %v", err)
		}
	}
	if destination2.Valid && len(destination2.String) > 0 {
		if err := parseDestination(destination2.String, &res.Destination2); err != nil {
			return nil, fmt.Errorf("could not parse destination2: %v", err)
		}
	}
	if destination3.Valid && len(destination3.String) > 0 {
		if err := parseDestination(destination3.String, &res.Destination3); err != nil {
			return nil, fmt.Errorf("could not parse destination3: %v", err)
		}
	}
	if destination4.Valid && len(destination4.String) > 0 {
		if err := parseDestination(destination4.String, &res.Destination4); err != nil {
			return nil, fmt.Errorf("could not parse destination4: %v", err)
		}
	}

	return res, nil
}

// OutdialTargetCreate insert a new outdialtarget record
func (h *handler) OutdialTargetCreate(ctx context.Context, t *outdialtarget.OutdialTarget) error {
	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. OutdialTargetCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(outdialtargetsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. OutdialTargetCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
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
	fields := commondatabasehandler.GetDBFields(&outdialtarget.OutdialTarget{})
	query, args, err := squirrel.
		Select(fields...).
		From(outdialtargetsTable).
		Where(squirrel.Eq{string(outdialtarget.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. outdialTargetGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. outdialTargetGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. outdialTargetGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. outdialTargetGetFromDB. id: %s, err: %v", id, err)
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

// OutdialTargetGets returns list of outdialtargets.
func (h *handler) OutdialTargetGets(ctx context.Context, token string, size uint64, filters map[outdialtarget.Field]any) ([]*outdialtarget.OutdialTarget, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&outdialtarget.OutdialTarget{})
	sb := squirrel.
		Select(fields...).
		From(outdialtargetsTable).
		Where(squirrel.Lt{string(outdialtarget.FieldTMCreate): token}).
		OrderBy(string(outdialtarget.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. OutdialTargetGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OutdialTargetGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*outdialtarget.OutdialTarget{}
	for rows.Next() {
		u, err := h.outdialTargetGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. OutdialTargetGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. OutdialTargetGets. err: %v", err)
	}

	return res, nil
}

// OutdialTargetUpdate updates the outdialtarget with given fields.
func (h *handler) OutdialTargetUpdate(ctx context.Context, id uuid.UUID, fields map[outdialtarget.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[outdialtarget.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutdialTargetUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(outdialtargetsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outdialtarget.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("OutdialTargetUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("OutdialTargetUpdate: exec failed: %w", err)
	}

	_ = h.outdialTargetUpdateToCache(ctx, id)
	return nil
}

// OutdialTargetDelete deletes the outdialtarget.
func (h *handler) OutdialTargetDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[outdialtarget.Field]any{
		outdialtarget.FieldTMUpdate: ts,
		outdialtarget.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutdialTargetDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(outdialtargetsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outdialtarget.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("OutdialTargetDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("could not execute. OutdialTargetDelete. err: %v", err)
	}

	_ = h.outdialTargetUpdateToCache(ctx, id)

	return nil
}

// OutdialTargetUpdateProgressing updates outdialtarget's basic info.
func (h *handler) OutdialTargetUpdateProgressing(ctx context.Context, id uuid.UUID, destinationIndex int) error {
	q := fmt.Sprintf(`
	update outdial_outdialtargets set
		try_count_%d = try_count_%d + 1,
		status = ?,
		tm_update = ?
	where
		id = ?
	`, destinationIndex, destinationIndex)

	if _, err := h.db.Exec(q, outdialtarget.StatusProgressing, h.utilHandler.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialTargetUpdateProgressing. err: %v", err)
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
	defer func() {
		_ = stmt.Close()
	}()

	// query
	rows, err := stmt.QueryContext(ctx, tryCount0, tryCount1, tryCount2, tryCount3, tryCount4, outdialID.Bytes(), limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetGetAvailable. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

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
