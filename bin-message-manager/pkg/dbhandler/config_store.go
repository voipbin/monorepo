package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"

	"monorepo/bin-message-manager/internal/configkeys"
)

var configColumns = []string{
	"config_key", "value_type", "value_json", "secret_ciphertext", "key_version", "revision", "updated_by", "tm_create", "tm_update",
}

// configStore implements confighandler.Store on the tables of this service. It has no cache:
// every read goes to the database, so a changed value is used by the next call on any replica.
type configStore struct {
	db *sql.DB
}

// NewConfigStore creates the store of the configs on the given database.
func NewConfigStore(db *sql.DB) confighandler.Store {
	return &configStore{db: db}
}

func nullString(b []byte) any {
	if len(b) == 0 {
		return nil
	}

	// a JSON column rejects a binary string, so the JSON is sent as text.
	return string(b)
}

func nullBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}

	return b
}

func nullUUID(u *uuid.UUID) any {
	if u == nil {
		return nil
	}

	return u.Bytes()
}

var timeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999 -0700 MST",
}

// parseTime accepts what the drivers return for a datetime column: a time.Time, or the text of
// the MySQL format (without parseTime) or of the sqlite format.
func parseTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case []byte:
		return parseTimeText(string(t))
	case string:
		return parseTimeText(t)
	}

	return time.Time{}, fmt.Errorf("unexpected time value of type %T", v)
}

func parseTimeText(s string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("unknown time format")
}

func scanConfigRow(rows *sql.Rows) (*confighandler.Row, error) {
	var (
		res                  confighandler.Row
		valueJSON, secret    []byte
		updatedBy            []byte
		tmCreate, tmUpdate   any
		keyVersion, revision int64
	)

	if err := rows.Scan(&res.Key, &res.ValueType, &valueJSON, &secret, &keyVersion, &revision, &updatedBy, &tmCreate, &tmUpdate); err != nil {
		return nil, fmt.Errorf("could not scan the config row: %w", err)
	}

	res.ValueJSON = valueJSON
	res.SecretCiphertext = secret
	res.KeyVersion = int(keyVersion)
	res.Revision = uint32(revision) //nolint:gosec // revision is an unsigned int column

	if len(updatedBy) == 16 {
		u := uuid.FromBytesOrNil(updatedBy)
		res.UpdatedBy = &u
	}

	var err error
	if res.TMCreate, err = parseTime(tmCreate); err != nil {
		return nil, fmt.Errorf("could not read tm_create of the config row: %w", err)
	}
	if res.TMUpdate, err = parseTime(tmUpdate); err != nil {
		return nil, fmt.Errorf("could not read tm_update of the config row: %w", err)
	}

	return &res, nil
}

// Get implements confighandler.Store.
func (s *configStore) Get(ctx context.Context, key string) (*confighandler.Row, error) {
	query, args, err := sq.Select(configColumns...).From(configkeys.Table).Where(sq.Eq{"config_key": key}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build the query. ConfigGet. err: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfigGet. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		if errRows := rows.Err(); errRows != nil {
			return nil, fmt.Errorf("could not iterate the rows. ConfigGet. err: %w", errRows)
		}
		return nil, confighandler.ErrNotFound
	}

	return scanConfigRow(rows)
}

// List implements confighandler.Store.
func (s *configStore) List(ctx context.Context) ([]*confighandler.Row, error) {
	query, args, err := sq.Select(configColumns...).From(configkeys.Table).OrderBy("config_key").ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build the query. ConfigList. err: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfigList. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*confighandler.Row{}
	for rows.Next() {
		row, errScan := scanConfigRow(rows)
		if errScan != nil {
			return nil, errScan
		}
		res = append(res, row)
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, fmt.Errorf("could not iterate the rows. ConfigList. err: %w", errRows)
	}

	return res, nil
}

// Insert implements confighandler.Store. A duplicate key is confighandler.ErrConflict.
func (s *configStore) Insert(ctx context.Context, row *confighandler.Row, hist *confighandler.History) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		query, args, err := sq.Insert(configkeys.Table).Columns(configColumns...).Values(
			row.Key, row.ValueType, nullString(row.ValueJSON), nullBytes(row.SecretCiphertext), row.KeyVersion, row.Revision,
			nullUUID(row.UpdatedBy), row.TMCreate.UTC(), row.TMUpdate.UTC(),
		).ToSql()
		if err != nil {
			return fmt.Errorf("could not build the query. ConfigInsert. err: %w", err)
		}

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			if isDuplicateKeyErr(err) {
				return confighandler.ErrConflict
			}
			return fmt.Errorf("could not execute. ConfigInsert. err: %w", err)
		}

		return insertHistory(ctx, tx, hist)
	})
}

// Update implements confighandler.Store. No matching row is confighandler.ErrConflict.
func (s *configStore) Update(ctx context.Context, row *confighandler.Row, expectRevision uint32, hist *confighandler.History) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		query, args, err := sq.Update(configkeys.Table).SetMap(map[string]any{
			"value_type":        row.ValueType,
			"value_json":        nullString(row.ValueJSON),
			"secret_ciphertext": nullBytes(row.SecretCiphertext),
			"key_version":       row.KeyVersion,
			"revision":          expectRevision + 1,
			"updated_by":        nullUUID(row.UpdatedBy),
			"tm_update":         row.TMUpdate.UTC(),
		}).Where(sq.Eq{"config_key": row.Key, "revision": expectRevision}).ToSql()
		if err != nil {
			return fmt.Errorf("could not build the query. ConfigUpdate. err: %w", err)
		}

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("could not execute. ConfigUpdate. err: %w", err)
		}
		if affected, errAffected := res.RowsAffected(); errAffected != nil {
			return fmt.Errorf("could not read the affected rows. ConfigUpdate. err: %w", errAffected)
		} else if affected == 0 {
			return confighandler.ErrConflict
		}

		return insertHistory(ctx, tx, hist)
	})
}

// Delete implements confighandler.Store. No matching row is confighandler.ErrConflict.
func (s *configStore) Delete(ctx context.Context, key string, expectRevision uint32, hist *confighandler.History) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		query, args, err := sq.Delete(configkeys.Table).Where(sq.Eq{"config_key": key, "revision": expectRevision}).ToSql()
		if err != nil {
			return fmt.Errorf("could not build the query. ConfigDelete. err: %w", err)
		}

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("could not execute. ConfigDelete. err: %w", err)
		}
		if affected, errAffected := res.RowsAffected(); errAffected != nil {
			return fmt.Errorf("could not read the affected rows. ConfigDelete. err: %w", errAffected)
		} else if affected == 0 {
			return confighandler.ErrConflict
		}

		return insertHistory(ctx, tx, hist)
	})
}

var historyColumns = []string{
	"id", "config_key", "action", "is_secret", "old_value_json", "new_value_json", "old_key_version", "new_key_version", "actor_id", "source", "tm_create",
}

func scanConfigHistory(rows *sql.Rows) (*confighandler.History, error) {
	var (
		res                    confighandler.History
		id, actor              []byte
		oldValue, newValue     []byte
		oldVersion, newVersion int64
		isSecret               bool
		tmCreate               any
	)

	if err := rows.Scan(&id, &res.Key, &res.Action, &isSecret, &oldValue, &newValue, &oldVersion, &newVersion, &actor, &res.Source, &tmCreate); err != nil {
		return nil, fmt.Errorf("could not scan the config history row: %w", err)
	}

	res.ID = uuid.FromBytesOrNil(id)
	res.ActorID = uuid.FromBytesOrNil(actor)
	res.IsSecret = isSecret
	res.OldValueJSON = oldValue
	res.NewValueJSON = newValue
	res.OldKeyVersion = int(oldVersion)
	res.NewKeyVersion = int(newVersion)

	var err error
	if res.TMCreate, err = parseTime(tmCreate); err != nil {
		return nil, fmt.Errorf("could not read tm_create of the config history row: %w", err)
	}

	return &res, nil
}

// historiesQuery builds the query of ListHistories. The condition after the cursor is written as
// an OR of two terms, so that the index (config_key, tm_create) is used.
func historiesQuery(key string, after confighandler.HistoryCursor, limit int) (string, []any, error) {
	builder := sq.Select(historyColumns...).From(configkeys.HistoryTable).Where(sq.Eq{"config_key": key})
	if !after.TMCreate.IsZero() {
		tm := after.TMCreate.UTC()
		builder = builder.Where(sq.Or{
			sq.Lt{"tm_create": tm},
			sq.And{sq.Eq{"tm_create": tm}, sq.Lt{"id": after.ID.Bytes()}},
		})
	}

	return builder.OrderBy("tm_create DESC", "id DESC").Limit(uint64(limit)).ToSql() //nolint:gosec // limit is positive here
}

// ListHistories implements confighandler.Store.
func (s *configStore) ListHistories(ctx context.Context, key string, after confighandler.HistoryCursor, limit int) ([]*confighandler.History, error) {
	if limit <= 0 {
		return []*confighandler.History{}, nil
	}

	query, args, err := historiesQuery(key, after, limit)
	if err != nil {
		return nil, fmt.Errorf("could not build the query. ConfigListHistories. err: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConfigListHistories. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*confighandler.History{}
	for rows.Next() {
		hist, errScan := scanConfigHistory(rows)
		if errScan != nil {
			return nil, errScan
		}
		res = append(res, hist)
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, fmt.Errorf("could not iterate the rows. ConfigListHistories. err: %w", errRows)
	}

	return res, nil
}

// inTx runs fn in a transaction that is committed when fn returns nil.
func (s *configStore) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin the transaction. err: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit the transaction. err: %w", err)
	}

	return nil
}

// insertHistory writes the history row. A secret or sensitive key has no value columns: the
// store writes exactly what the library gave, and the table has a CHECK for the same rule.
func insertHistory(ctx context.Context, tx *sql.Tx, hist *confighandler.History) error {
	if hist == nil {
		return errors.New("the history row is required")
	}

	query, args, err := sq.Insert(configkeys.HistoryTable).Columns(
		"id", "config_key", "action", "is_secret", "old_value_json", "new_value_json", "old_key_version", "new_key_version", "actor_id", "source", "tm_create",
	).Values(
		hist.ID.Bytes(), hist.Key, hist.Action, hist.IsSecret, nullString(hist.OldValueJSON), nullString(hist.NewValueJSON),
		hist.OldKeyVersion, hist.NewKeyVersion, hist.ActorID.Bytes(), hist.Source, hist.TMCreate.UTC(),
	).ToSql()
	if err != nil {
		return fmt.Errorf("could not build the query. ConfigHistoryInsert. err: %w", err)
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. ConfigHistoryInsert. err: %w", err)
	}

	return nil
}
