package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-tag-manager/models/tag"
)

const (
	tagTable = "tag_tags"
)

// tagGetFromRow scans a single row into a Tag struct using db tags
func (h *handler) tagGetFromRow(rows *sql.Rows) (*tag.Tag, error) {
	res := &tag.Tag{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. tagGetFromRow. err: %v", err)
	}

	return res, nil
}

// TagCreate creates new tag record
func (h *handler) TagCreate(ctx context.Context, t *tag.Tag) error {
	t.TMCreate = h.utilHandler.TimeGetCurTime()
	t.TMUpdate = DefaultTimeStamp
	t.TMDelete = DefaultTimeStamp

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. TagCreate. err: %v", err)
	}

	query, args, err := sq.Insert(tagTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TagCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. TagCreate. err: %v", err)
	}

	// update the cache
	_ = h.tagUpdateToCache(ctx, t.ID)

	return nil
}

// tagUpdateToCache gets the tag from the DB and updates the cache.
func (h *handler) tagUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.tagGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.tagSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// tagSetToCache sets the given tag to the cache
func (h *handler) tagSetToCache(ctx context.Context, t *tag.Tag) error {
	if err := h.cache.TagSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// tagGetFromCache returns tag from the cache.
func (h *handler) tagGetFromCache(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	res, err := h.cache.TagGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// tagGetFromDB returns tag from the DB.
func (h *handler) tagGetFromDB(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&tag.Tag{})

	query, args, err := sq.Select(columns...).
		From(tagTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. tagGetFromDB. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. tagGetFromDB. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.tagGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. tagGetFromDB. err: %v", err)
	}

	return res, nil
}

// TagGet returns tag.
func (h *handler) TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	res, err := h.tagGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.tagGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.tagSetToCache(ctx, res)

	return res, nil
}

// TagGets returns tags based on filters.
func (h *handler) TagGets(ctx context.Context, size uint64, token string, filters map[tag.Field]any) ([]*tag.Tag, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&tag.Tag{})

	builder := sq.Select(columns...).
		From(tagTable).
		Where("tm_create < ?", token).
		OrderBy("tm_create desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. TagGets. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. TagGets. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TagGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []*tag.Tag
	for rows.Next() {
		t, err := h.tagGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. TagGets. err: %v", err)
		}

		res = append(res, t)
	}

	return res, nil
}

// TagUpdate updates a tag with the given fields.
func (h *handler) TagUpdate(ctx context.Context, id uuid.UUID, fields map[tag.Field]any) error {
	// add update timestamp
	fields[tag.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. TagUpdate. err: %v", err)
	}

	query, args, err := sq.Update(tagTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TagUpdate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. TagUpdate. err: %v", err)
	}

	// update the cache
	_ = h.tagUpdateToCache(ctx, id)

	return nil
}

// TagSetBasicInfo sets the tag's basic info.
func (h *handler) TagSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	fields := map[tag.Field]any{
		tag.FieldName:   name,
		tag.FieldDetail: detail,
	}

	return h.TagUpdate(ctx, id, fields)
}

// TagDelete deletes the tag info.
func (h *handler) TagDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()
	fields := map[tag.Field]any{
		tag.FieldTMUpdate: ts,
		tag.FieldTMDelete: ts,
	}

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. TagDelete. err: %v", err)
	}

	query, args, err := sq.Update(tagTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TagDelete. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. TagDelete. err: %v", err)
	}

	// update the cache
	_ = h.tagUpdateToCache(ctx, id)

	return nil
}
