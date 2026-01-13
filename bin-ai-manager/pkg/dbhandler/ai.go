package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/ai"
)

const (
	aiTable = "ai_ais"
)

// AICreate creates new ai record.
func (h *handler) AICreate(ctx context.Context, c *ai.AI) error {
	c.TMCreate = h.utilHandler.TimeGetCurTime()
	c.TMUpdate = DefaultTimeStamp
	c.TMDelete = DefaultTimeStamp

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("AICreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(aiTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AICreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AICreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, c.ID)

	return nil
}

// aiGetFromCache returns ai from the cache.
func (h *handler) aiGetFromCache(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, err := h.cache.AIGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// aiGetFromDB returns ai from the DB.
func (h *handler) aiGetFromDB(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	cols := commondatabasehandler.GetDBFields(ai.AI{})

	query, args, err := sq.Select(cols...).
		From(aiTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("aiGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("aiGetFromDB: could not query. err: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &ai.AI{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("aiGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// aiUpdateToCache gets the ai from the DB and updates the cache.
func (h *handler) aiUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.aiGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.aiSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// aiSetToCache sets the given ai to the cache.
func (h *handler) aiSetToCache(ctx context.Context, c *ai.AI) error {
	if err := h.cache.AISet(ctx, c); err != nil {
		return err
	}

	return nil
}

// AIGet returns ai.
func (h *handler) AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, err := h.aiGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.aiGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.aiSetToCache(ctx, res)

	return res, nil
}

// AIDelete deletes the ai.
func (h *handler) AIDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	query, args, err := sq.Update(aiTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, id)

	return nil
}

// AIGets returns a list of ais.
func (h *handler) AIGets(ctx context.Context, size uint64, token string, filters map[ai.Field]any) ([]*ai.AI, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(ai.AI{})

	builder := sq.Select(cols...).
		From(aiTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AIGets: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIGets: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIGets: could not query. err: %v", err)
	}
	defer rows.Close()

	res := []*ai.AI{}
	for rows.Next() {
		u := &ai.AI{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("AIGets: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// AIUpdate updates the ai fields.
func (h *handler) AIUpdate(ctx context.Context, id uuid.UUID, fields map[ai.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeGetCurTime()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("AIUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(aiTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, id)

	return nil
}
