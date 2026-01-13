package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-conversation-manager/models/media"
)

var (
	mediasTable = "conversation_medias"
)

// mediaGetFromRow gets the media from the row.
func (h *handler) mediaGetFromRow(row *sql.Rows) (*media.Media, error) {
	res := &media.Media{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. mediaGetFromRow. err: %v", err)
	}

	return res, nil
}

// MediaCreate creates a new media record
func (h *handler) MediaCreate(ctx context.Context, m *media.Media) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	m.TMCreate = now
	m.TMUpdate = commondatabasehandler.DefaultTimeStamp
	m.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(m)
	if err != nil {
		return fmt.Errorf("could not prepare fields. MediaCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(mediasTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. MediaCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. MediaCreate. err: %v", err)
	}

	_ = h.mediaUpdateToCache(ctx, m.ID)

	return nil
}

// mediaGetFromDB gets the media info from the db.
func (h *handler) mediaGetFromDB(ctx context.Context, id uuid.UUID) (*media.Media, error) {
	fields := commondatabasehandler.GetDBFields(&media.Media{})
	query, args, err := squirrel.
		Select(fields...).
		From(mediasTable).
		Where(squirrel.Eq{"id": id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. mediaGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not query. mediaGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.mediaGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// mediaUpdateToCache gets the media from the DB and update the cache.
func (h *handler) mediaUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.mediaGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.mediaSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// mediaSetToCache sets the given media to the cache
func (h *handler) mediaSetToCache(ctx context.Context, flow *media.Media) error {
	if err := h.cache.MediaSet(ctx, flow); err != nil {
		return err
	}

	return nil
}

// mediaGetFromCache returns media from the cache.
func (h *handler) mediaGetFromCache(ctx context.Context, id uuid.UUID) (*media.Media, error) {

	// get from cache
	res, err := h.cache.MediaGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MediaGet returns media.
func (h *handler) MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error) {

	res, err := h.mediaGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.mediaGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.mediaSetToCache(ctx, res)

	return res, nil
}
