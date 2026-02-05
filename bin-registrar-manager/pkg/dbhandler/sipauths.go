package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-registrar-manager/models/sipauth"
)

const (
	sipauthsTable = "registrar_sip_auths"
)

// sipauthGetFromRow gets the sipauth from the row
func (h *handler) sipauthGetFromRow(row *sql.Rows) (*sipauth.SIPAuth, error) {
	res := &sipauth.SIPAuth{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. sipauthGetFromRow. err: %v", err)
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

// SIPAuthCreate creates new SIPAuth record.
func (h *handler) SIPAuthCreate(ctx context.Context, t *sipauth.SIPAuth) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	t.TMCreate = now
	t.TMUpdate = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. SIPAuthCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(sipauthsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. SIPAuthCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. SIPAuthCreate. err: %v", err)
	}

	return nil
}

// SIPAuthUpdate updates sipauth record with given fields.
func (h *handler) SIPAuthUpdate(ctx context.Context, id uuid.UUID, fields map[sipauth.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[sipauth.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("SIPAuthUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(sipauthsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(sipauth.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("SIPAuthUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("SIPAuthUpdate: exec failed: %w", err)
	}

	return nil
}

// SIPAuthDelete deletes the sip auth
func (h *handler) SIPAuthDelete(ctx context.Context, id uuid.UUID) error {
	sb := squirrel.Delete(sipauthsTable).
		Where(squirrel.Eq{string(sipauth.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("SIPAuthDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("SIPAuthDelete: exec failed: %w", err)
	}

	return nil
}

// SIPAuthGet returns SIPAuth.
func (h *handler) SIPAuthGet(ctx context.Context, id uuid.UUID) (*sipauth.SIPAuth, error) {
	fields := commondatabasehandler.GetDBFields(&sipauth.SIPAuth{})
	query, args, err := squirrel.
		Select(fields...).
		From(sipauthsTable).
		Where(squirrel.Eq{string(sipauth.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. SIPAuthGet. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. SIPAuthGet. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. SIPAuthGet. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.sipauthGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. SIPAuthGet. id: %s, err: %v", id, err)
	}

	return res, nil
}
