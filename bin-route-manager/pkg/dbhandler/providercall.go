package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-route-manager/models/providercall"
)

const (
	providerCallsTable = "route_providercalls"
)

// providerCallGetFromRow scans one row into a ProviderCall.
func (h *handler) providerCallGetFromRow(row *sql.Rows) (*providercall.ProviderCall, error) {
	res := &providercall.ProviderCall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. providerCallGetFromRow. err: %v", err)
	}

	return res, nil
}

// ProviderCallCreate inserts a new providercall record.
func (h *handler) ProviderCallCreate(ctx context.Context, p *providercall.ProviderCall) error {
	now := h.utilHandler.TimeNow()

	p.TMCreate = now
	p.TMUpdate = nil
	p.TMDelete = nil

	// initialize slice fields if nil to keep JSON columns non-null
	if p.Destinations == nil {
		p.Destinations = []commonaddress.Address{}
	}
	if p.CallIDs == nil {
		p.CallIDs = []uuid.UUID{}
	}
	if p.GroupcallIDs == nil {
		p.GroupcallIDs = []uuid.UUID{}
	}

	fields, err := commondatabasehandler.PrepareFields(p)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ProviderCallCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(providerCallsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ProviderCallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ProviderCallCreate. err: %v", err)
	}

	return nil
}

// providerCallGetFromDB reads the providercall from the db.
func (h *handler) providerCallGetFromDB(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error) {
	fields := commondatabasehandler.GetDBFields(&providercall.ProviderCall{})
	query, args, err := squirrel.
		Select(fields...).
		From(providerCallsTable).
		Where(squirrel.Eq{string(providercall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. providerCallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. providerCallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. providerCallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.providerCallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. providerCallGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// ProviderCallGet returns the providercall for the given id.
// ProviderCall records are not cached because they are admin-audit data,
// not hot-path resources.
func (h *handler) ProviderCallGet(ctx context.Context, id uuid.UUID) (*providercall.ProviderCall, error) {
	return h.providerCallGetFromDB(ctx, id)
}

// ProviderCallList returns a paginated list of providercalls matching the filters.
func (h *handler) ProviderCallList(ctx context.Context, token string, limit uint64, filters map[providercall.Field]any) ([]*providercall.ProviderCall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&providercall.ProviderCall{})
	sb := squirrel.
		Select(fields...).
		From(providerCallsTable).
		Where(squirrel.Eq{string(providercall.FieldTMDelete): nil}).
		Where(squirrel.Lt{string(providercall.FieldTMCreate): token}).
		OrderBy(string(providercall.FieldTMCreate)+" DESC", string(providercall.FieldID)+" DESC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ProviderCallList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ProviderCallList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ProviderCallList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*providercall.ProviderCall{}
	for rows.Next() {
		u, err := h.providerCallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ProviderCallList. err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ProviderCallList. err: %v", err)
	}

	return res, nil
}

// ProviderCallDelete soft-deletes the given providercall.
func (h *handler) ProviderCallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[providercall.Field]any{
		providercall.FieldTMUpdate: ts,
		providercall.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ProviderCallDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(providerCallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(providercall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ProviderCallDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("ProviderCallDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
