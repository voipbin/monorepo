package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-campaign-manager/models/outplan"
)

const (
	outplansTable = "campaign_outplans"
)

// outplanGetFromRow gets the outplan from the row.
func (h *handler) outplanGetFromRow(row *sql.Rows) (*outplan.Outplan, error) {
	res := &outplan.Outplan{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. outplanGetFromRow. err: %v", err)
	}

	return res, nil
}

// OutplanCreate insert a new plan record
func (h *handler) OutplanCreate(ctx context.Context, o *outplan.Outplan) error {
	now := h.util.TimeNow()

	// Set timestamps
	o.TMCreate = now
	o.TMUpdate = nil
	o.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(o)
	if err != nil {
		return fmt.Errorf("could not prepare fields. OutplanCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(outplansTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. OutplanCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. OutplanCreate. err: %v", err)
	}

	_ = h.outplanUpdateToCache(ctx, o.ID)

	return nil
}

// outplanUpdateToCache gets the outplan from the DB and update the cache.
func (h *handler) outplanUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.outplanGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.outplanSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// outplanSetToCache sets the given outplan to the cache
func (h *handler) outplanSetToCache(ctx context.Context, f *outplan.Outplan) error {
	if err := h.cache.OutplanSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// outplanGetFromCache returns outplan from the cache if possible.
func (h *handler) outplanGetFromCache(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {

	// get from cache
	res, err := h.cache.OutplanGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outplanGetFromDB gets the outplan info from the db.
func (h *handler) outplanGetFromDB(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {
	fields := commondatabasehandler.GetDBFields(&outplan.Outplan{})
	query, args, err := squirrel.
		Select(fields...).
		From(outplansTable).
		Where(squirrel.Eq{string(outplan.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. outplanGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. outplanGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. outplanGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.outplanGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OutplanDelete deletes the given outplan
func (h *handler) OutplanDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.util.TimeNow()

	fields := map[outplan.Field]any{
		outplan.FieldTMUpdate: ts,
		outplan.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutplanDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(outplansTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outplan.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("OutplanDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("OutplanDelete: exec failed: %w", err)
	}

	// update cache
	_ = h.outplanUpdateToCache(ctx, id)

	return nil
}

// OutplanGet returns outplan.
func (h *handler) OutplanGet(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {

	res, err := h.outplanGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.outplanGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.outplanSetToCache(ctx, res)

	return res, nil
}

// OutplanGets returns list of outplans with filters.
func (h *handler) OutplanList(ctx context.Context, token string, size uint64, filters map[outplan.Field]any) ([]*outplan.Outplan, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&outplan.Outplan{})
	sb := squirrel.
		Select(fields...).
		From(outplansTable).
		Where(squirrel.Lt{string(outplan.FieldTMCreate): token}).
		OrderBy(string(outplan.FieldTMCreate) + " DESC", string(outplan.FieldID) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. OutplanList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OutplanList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutplanList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*outplan.Outplan{}
	for rows.Next() {
		u, err := h.outplanGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. OutplanGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. OutplanList. err: %v", err)
	}

	return res, nil
}

// OutplanGetsByCustomerID returns list of outplans.
func (h *handler) OutplanListByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outplan.Outplan, error) {
	filters := map[outplan.Field]any{
		outplan.FieldCustomerID: customerID,
		outplan.FieldDeleted:    false,
	}

	return h.OutplanList(ctx, token, limit, filters)
}

// OutplanUpdate updates outplan fields.
func (h *handler) OutplanUpdate(ctx context.Context, id uuid.UUID, fields map[outplan.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[outplan.FieldTMUpdate] = h.util.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutplanUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(outplansTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outplan.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("OutplanUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("OutplanUpdate: exec failed: %w", err)
	}

	_ = h.outplanUpdateToCache(ctx, id)
	return nil
}

// OutplanUpdateBasicInfo updates outplan's basic information.
func (h *handler) OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	fields := map[outplan.Field]any{
		outplan.FieldName:   name,
		outplan.FieldDetail: detail,
	}

	return h.OutplanUpdate(ctx, id, fields)
}

// OutplanUpdateDialInfo updates outplan's dial related information.
func (h *handler) OutplanUpdateDialInfo(
	ctx context.Context,
	id uuid.UUID,
	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) error {
	fields := map[outplan.Field]any{
		outplan.FieldSource:       source,
		outplan.FieldDialTimeout:  dialTimeout,
		outplan.FieldTryInterval:  tryInterval,
		outplan.FieldMaxTryCount0: maxTryCount0,
		outplan.FieldMaxTryCount1: maxTryCount1,
		outplan.FieldMaxTryCount2: maxTryCount2,
		outplan.FieldMaxTryCount3: maxTryCount3,
		outplan.FieldMaxTryCount4: maxTryCount4,
	}

	return h.OutplanUpdate(ctx, id, fields)
}
