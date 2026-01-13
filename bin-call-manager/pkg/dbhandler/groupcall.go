package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/groupcall"
)

var (
	groupcallTable = "call_groupcalls"
)

// groupcallGetFromRow gets the groupcall from the row.
func (h *handler) groupcallGetFromRow(row *sql.Rows) (*groupcall.Groupcall, error) {
	res := &groupcall.Groupcall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. groupcallGetFromRow. err: %v", err)
	}

	// Initialize nil values to defaults
	if res.Source == nil {
		res.Source = &commonaddress.Address{}
	}
	if res.Destinations == nil {
		res.Destinations = []commonaddress.Address{}
	}
	if res.CallIDs == nil {
		res.CallIDs = []uuid.UUID{}
	}
	if res.GroupcallIDs == nil {
		res.GroupcallIDs = []uuid.UUID{}
	}

	return res, nil
}

// GroupcallCreate sets groupcall.
func (h *handler) GroupcallCreate(ctx context.Context, c *groupcall.Groupcall) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = commondatabasehandler.DefaultTimeStamp
	c.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Initialize nil slices/pointers
	if c.Source == nil {
		c.Source = &commonaddress.Address{}
	}
	if c.Destinations == nil {
		c.Destinations = []commonaddress.Address{}
	}
	if c.CallIDs == nil {
		c.CallIDs = []uuid.UUID{}
	}
	if c.GroupcallIDs == nil {
		c.GroupcallIDs = []uuid.UUID{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. GroupcallCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(groupcallTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. GroupcallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "could not execute. GroupcallCreate.")
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, c.ID)

	return nil
}

// GroupcallGet returns groupcall.
func (h *handler) GroupcallGet(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	res, err := h.groupcallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.groupcallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.groupcallSetToCache(ctx, res)

	return res, nil
}

// GroupcallGets returns a list of groupcalls.
func (h *handler) GroupcallGets(ctx context.Context, size uint64, token string, filters map[groupcall.Field]any) ([]*groupcall.Groupcall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	dbFields := commondatabasehandler.GetDBFields(&groupcall.Groupcall{})
	sb := squirrel.
		Select(dbFields...).
		From(groupcallTable).
		Where(squirrel.Lt{string(groupcall.FieldTMCreate): token}).
		OrderBy(string(groupcall.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. GroupcallGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. GroupcallGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. GroupcallGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*groupcall.Groupcall{}
	for rows.Next() {
		u, err := h.groupcallGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrap(err, "Could not get data. GroupcallGets.")
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. GroupcallGets. err: %v", err)
	}

	return res, nil
}

// GroupcallUpdate updates groupcall fields using a generic typed field map
func (h *handler) GroupcallUpdate(ctx context.Context, id uuid.UUID, fields map[groupcall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[groupcall.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("GroupcallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(groupcallTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(groupcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("GroupcallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("GroupcallUpdate: exec failed: %w", err)
	}

	_ = h.groupcallUpdateToCache(ctx, id)
	return nil
}

// GroupcallSetAnswerCallID updates the answer call id.
func (h *handler) GroupcallSetAnswerCallID(ctx context.Context, id uuid.UUID, answerCallID uuid.UUID) error {
	return h.GroupcallUpdate(ctx, id, map[groupcall.Field]any{
		groupcall.FieldAnswerCallID: answerCallID,
	})
}

// GroupcallSetAnswerGroupcallID updates the answer groupcall id.
func (h *handler) GroupcallSetAnswerGroupcallID(ctx context.Context, id uuid.UUID, answerGroupcallID uuid.UUID) error {
	return h.GroupcallUpdate(ctx, id, map[groupcall.Field]any{
		groupcall.FieldAnswerGroupcallID: answerGroupcallID,
	})
}

// groupcallGetFromCache returns groupcall from the cache.
func (h *handler) groupcallGetFromCache(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	res, err := h.cache.GroupcallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// groupcallGetFromDB returns groupcall from the DB.
func (h *handler) groupcallGetFromDB(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	fields := commondatabasehandler.GetDBFields(&groupcall.Groupcall{})
	query, args, err := squirrel.
		Select(fields...).
		From(groupcallTable).
		Where(squirrel.Eq{string(groupcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. groupcallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. groupcallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. groupcallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.groupcallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. groupcallGetFromDB, err: %v", err)
	}

	return res, nil
}

// groupcallUpdateToCache gets the groupcall from the DB and update the cache.
func (h *handler) groupcallUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.groupcallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.groupcallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// groupcallSetToCache sets the given groupcall to the cache
func (h *handler) groupcallSetToCache(ctx context.Context, data *groupcall.Groupcall) error {
	if err := h.cache.GroupcallSet(ctx, data); err != nil {
		return err
	}
	return nil
}

// GroupcallDelete deletes the groupcall
func (h *handler) GroupcallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()
	return h.GroupcallUpdate(ctx, id, map[groupcall.Field]any{
		groupcall.FieldTMUpdate: ts,
		groupcall.FieldTMDelete: ts,
	})
}

// GroupcallDecreaseCallCount decreases the call count
func (h *handler) GroupcallDecreaseCallCount(ctx context.Context, id uuid.UUID) error {
	// Use raw SQL for atomic decrement operation
	q := `
	update call_groupcalls set
		call_count = call_count - 1,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallDecreaseCallCount. err: %v", err)
	}

	_ = h.groupcallUpdateToCache(ctx, id)
	return nil
}

// GroupcallDecreaseGroupcallCount decreases the groupcall count
func (h *handler) GroupcallDecreaseGroupcallCount(ctx context.Context, id uuid.UUID) error {
	// Use raw SQL for atomic decrement operation
	q := `
	update call_groupcalls set
		groupcall_count = groupcall_count - 1,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallDecreaseGroupcallCount. err: %v", err)
	}

	_ = h.groupcallUpdateToCache(ctx, id)
	return nil
}

// GroupcallSetStatus updates the status
func (h *handler) GroupcallSetStatus(ctx context.Context, id uuid.UUID, status groupcall.Status) error {
	return h.GroupcallUpdate(ctx, id, map[groupcall.Field]any{
		groupcall.FieldStatus: status,
	})
}

// GroupcallSetCallIDsAndCallCountAndDialIndex updates the call_ids and call_count and dial_index
func (h *handler) GroupcallSetCallIDsAndCallCountAndDialIndex(ctx context.Context, id uuid.UUID, callIDs []uuid.UUID, callCount int, dialIndex int) error {
	if callIDs == nil {
		callIDs = []uuid.UUID{}
	}
	tmpCallIDs, err := json.Marshal(callIDs)
	if err != nil {
		return errors.Wrap(err, "could not marshal the call_ids. GroupcallSetCallIDsAndCallCountAndDialIndex.")
	}

	q := `
	update call_groupcalls set
		call_ids = ?,
		call_count = ?,
		dial_index = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, tmpCallIDs, callCount, dialIndex, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetCallIDsAndCallCountAndDialIndex. err: %v", err)
	}

	_ = h.groupcallUpdateToCache(ctx, id)
	return nil
}

// GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex updates the groupcall_ids and groupcall_count and dial_index
func (h *handler) GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx context.Context, id uuid.UUID, groupcallIDs []uuid.UUID, groupcallCount int, dialIndex int) error {
	if groupcallIDs == nil {
		groupcallIDs = []uuid.UUID{}
	}
	tmpGroupcallIDs, err := json.Marshal(groupcallIDs)
	if err != nil {
		return errors.Wrap(err, "could not marshal the groupcall_ids. GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex.")
	}

	q := `
	update call_groupcalls set
		groupcall_ids = ?,
		groupcall_count = ?,
		dial_index = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, tmpGroupcallIDs, groupcallCount, dialIndex, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex. err: %v", err)
	}

	_ = h.groupcallUpdateToCache(ctx, id)
	return nil
}
