package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
)

const (
	trunksTable = "registrar_trunks"
)

// trunkGetFromRow gets the trunk from the row
func (h *handler) trunkGetFromRow(row *sql.Rows) (*trunk.Trunk, error) {
	res := &trunk.Trunk{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. trunkGetFromRow. err: %v", err)
	}

	// initialize nil slices to empty slices
	if res.AuthTypes == nil {
		res.AuthTypes = []sipauth.AuthType{}
	}
	if res.AllowedIPs == nil {
		res.AllowedIPs = []string{}
	}

	return res, nil
}

// TrunkCountByCustomerID returns the count of active trunks for the given customer.
func (h *handler) TrunkCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	query, args, err := squirrel.
		Select("COUNT(*)").
		From(trunksTable).
		Where(squirrel.Eq{string(trunk.FieldCustomerID): customerID.Bytes()}).
		Where(squirrel.Eq{string(trunk.FieldTMDelete): nil}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("TrunkCountByCustomerID: could not build query. err: %v", err)
	}

	var count int
	if err := h.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("TrunkCountByCustomerID: could not query. err: %v", err)
	}

	return count, nil
}

// TrunkCreate creates new Trunk record.
func (h *handler) TrunkCreate(ctx context.Context, t *trunk.Trunk) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	t.TMCreate = now
	t.TMUpdate = nil
	t.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. TrunkCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(trunksTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TrunkCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. TrunkCreate. err: %v", err)
	}

	// update the cache
	_ = h.trunkUpdateToCache(ctx, t.ID)

	return nil
}

// trunkGetFromDB returns Trunk from the DB.
func (h *handler) trunkGetFromDB(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {
	fields := commondatabasehandler.GetDBFields(&trunk.Trunk{})
	query, args, err := squirrel.
		Select(fields...).
		From(trunksTable).
		Where(squirrel.Eq{string(trunk.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. trunkGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. trunkGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. trunkGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.trunkGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. trunkGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// trunkUpdateToCache gets the trunk from the DB and update the cache.
func (h *handler) trunkUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.trunkGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.trunkSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// trunkSetToCache sets the given trunk to the cache
func (h *handler) trunkSetToCache(ctx context.Context, e *trunk.Trunk) error {
	if err := h.cache.TrunkSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// trunkGetFromCache returns trunk from the cache.
func (h *handler) trunkGetFromCache(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {

	// get from cache
	res, err := h.cache.TrunkGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// trunkGetByDomainNameFromCache returns Domain from the cache.
func (h *handler) trunkGetByDomainNameFromCache(ctx context.Context, domainName string) (*trunk.Trunk, error) {

	// get from cache
	res, err := h.cache.TrunkGetByDomainName(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// trunkDeleteFromCache deletes Domain from the cache.
//
//nolint:unused // good to have. will use in the future
func (h *handler) trunkDeleteFromCache(ctx context.Context, id uuid.UUID, name string) error {

	// get from cache
	if err := h.cache.TrunkDel(ctx, id, name); err != nil {
		return err
	}

	return nil
}

// TrunkUpdate updates trunk record with given fields.
func (h *handler) TrunkUpdate(ctx context.Context, id uuid.UUID, fields map[trunk.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[trunk.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("TrunkUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(trunksTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(trunk.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("TrunkUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("TrunkUpdate: exec failed: %w", err)
	}

	_ = h.trunkUpdateToCache(ctx, id)
	return nil
}

// TrunkGet returns Trunk.
func (h *handler) TrunkGet(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {

	res, err := h.trunkGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.trunkGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.trunkSetToCache(ctx, res)

	return res, nil
}

// TrunkGetByDomainName returns Trunk of the given domain name.
func (h *handler) TrunkGetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error) {

	res, err := h.trunkGetByDomainNameFromCache(ctx, domainName)
	if err == nil {
		// Check if cached trunk is deleted (soft delete check)
		if res.TMDelete == nil {
			return res, nil
		}
		// Cached trunk is deleted, treat as not found
		res = nil
	}

	filters := map[trunk.Field]any{
		trunk.FieldDomainName: domainName,
		trunk.FieldDeleted:    false,
	}

	tmp, err := h.TrunkList(ctx, 1, "", filters)
	if err != nil {
		return nil, err
	}

	if len(tmp) == 0 {
		return nil, ErrNotFound
	}

	res = tmp[0]

	// set to the cache
	_ = h.trunkSetToCache(ctx, res)

	return res, nil
}

// TrunkList returns trunks.
func (h *handler) TrunkList(ctx context.Context, size uint64, token string, filters map[trunk.Field]any) ([]*trunk.Trunk, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&trunk.Trunk{})
	sb := squirrel.
		Select(fields...).
		From(trunksTable).
		Where(squirrel.Lt{string(trunk.FieldTMCreate): token}).
		OrderBy(string(trunk.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. TrunkGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. TrunkGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TrunkGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*trunk.Trunk{}
	for rows.Next() {
		u, err := h.trunkGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. TrunkGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. TrunkGets. err: %v", err)
	}

	return res, nil
}

// TrunkDelete deletes given Trunk
func (h *handler) TrunkDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[trunk.Field]any{
		trunk.FieldTMUpdate: ts,
		trunk.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("TrunkDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(trunksTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(trunk.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("TrunkDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("TrunkDelete: exec failed: %w", err)
	}

	_ = h.trunkUpdateToCache(ctx, id)

	return nil
}
