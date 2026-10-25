package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	gomysql "github.com/go-sql-driver/mysql"
	pkgerrors "github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
)

const (
	providersTable = "number_providers"

	// mysqlErrDupEntry is MySQL's ER_DUP_ENTRY errno, raised on a UNIQUE/PK constraint violation.
	mysqlErrDupEntry = 1062
)

// isDuplicateKeyErr reports whether the error is a duplicate-key violation of MySQL (typed
// errno 1062) or of sqlite (used by the test harness).
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}

	var myErr *gomysql.MySQLError
	if errors.As(err, &myErr) && myErr.Number == mysqlErrDupEntry {
		return true
	}

	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (h *handler) providerGetFromRow(row *sql.Rows) (*provider.Provider, error) {
	res := &provider.Provider{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. providerGetFromRow. err: %w", err)
	}

	return res, nil
}

// NumberProviderGet returns the provider setting row. It returns ErrNotFound if there is no row.
func (h *handler) NumberProviderGet(ctx context.Context, name number.ProviderName) (*provider.Provider, error) {
	fields := commondatabasehandler.GetDBFields(&provider.Provider{})
	query, args, err := squirrel.
		Select(fields...).
		From(providersTable).
		Where(squirrel.Eq{string(provider.FieldProviderName): string(name)}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. NumberProviderGet. err: %w", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberProviderGet. err: %w", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if errRow := row.Err(); errRow != nil {
			return nil, fmt.Errorf("row iteration error. NumberProviderGet. err: %w", errRow)
		}
		return nil, ErrNotFound
	}

	res, err := h.providerGetFromRow(row)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "could not get the provider. NumberProviderGet. name: %s", name)
	}

	return res, nil
}

// NumberProviderEnsure inserts the given row if no row with the same provider name exists.
// An existing row is left untouched (a duplicate key is treated as success), so concurrent
// callers never overwrite each other.
func (h *handler) NumberProviderEnsure(ctx context.Context, p *provider.Provider) error {
	p.TMCreate = h.utilHandler.TimeNow()
	p.TMUpdate = nil

	fields, err := commondatabasehandler.PrepareFields(p)
	if err != nil {
		return fmt.Errorf("could not prepare fields. NumberProviderEnsure. err: %w", err)
	}

	query, args, err := squirrel.
		Insert(providersTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. NumberProviderEnsure. err: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		if isDuplicateKeyErr(err) {
			return nil
		}
		return fmt.Errorf("could not execute query. NumberProviderEnsure. err: %w", err)
	}

	return nil
}

// NumberProviderUpdate updates only the given columns of the provider row. It returns
// ErrNotFound if there is no such row.
func (h *handler) NumberProviderUpdate(ctx context.Context, name number.ProviderName, fields map[provider.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[provider.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("NumberProviderUpdate: prepare fields failed: %w", err)
	}

	sqlStr, args, err := squirrel.Update(providersTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(provider.FieldProviderName): string(name)}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("NumberProviderUpdate: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("NumberProviderUpdate: exec failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.Wrap(err, "could not get rows affected")
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
