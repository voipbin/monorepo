package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

var (
	pipecatcallsTable  = "pipecat_pipecatcalls"
	pipecatcallsFields = []string{
		string(pipecatcall.FieldID),
		string(pipecatcall.FieldCustomerID),

		string(pipecatcall.FieldActiveflowID),
		string(pipecatcall.FieldReferenceType),
		string(pipecatcall.FieldReferenceID),

		string(pipecatcall.FieldHostID),

		string(pipecatcall.FieldLLMType),
		string(pipecatcall.FieldSTTType),
		string(pipecatcall.FieldTTSType),
		string(pipecatcall.FieldTTSVoiceID),
		string(pipecatcall.FieldLLMMessages),

		string(pipecatcall.FieldTMCreate),
		string(pipecatcall.FieldTMUpdate),
		string(pipecatcall.FieldTMDelete),
	}
)

func (h *handler) pipecatcallGetFromRow(row *sql.Rows) (*pipecatcall.Pipecatcall, error) {
	var messages string

	res := &pipecatcall.Pipecatcall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ActiveflowID,
		&res.ReferenceType,
		&res.ReferenceID,

		&res.HostID,

		&res.LLMType,
		&res.STTType,
		&res.TTSType,
		&res.TTSVoiceID,
		&messages,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. pipecatcallGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(messages), &res.LLMMessages); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. PipecatcallGet. err: %v", err)
	}

	return res, nil
}

func (h *handler) PipecatcallCreate(ctx context.Context, f *pipecatcall.Pipecatcall) error {
	now := h.utilHandler.TimeGetCurTime()

	tmpMessages, err := json.Marshal(f.LLMMessages)
	if err != nil {
		return fmt.Errorf("could not marshal messages. PipecatcallCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(pipecatcallsTable).
		Columns(pipecatcallsFields...).
		Values(
			f.ID.Bytes(),
			f.CustomerID.Bytes(),

			f.ActiveflowID.Bytes(),
			f.ReferenceType,
			f.ReferenceID.Bytes(),

			f.HostID,

			f.LLMType,
			f.STTType,
			f.TTSType,
			f.TTSVoiceID,
			tmpMessages,

			now,                                    // tm_create
			commondatabasehandler.DefaultTimeStamp, // tm_update
			commondatabasehandler.DefaultTimeStamp, // tm_delete
		).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. PipecatcallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. PipecatcallCreate. err: %v", err)
	}

	_ = h.pipecatcallUpdateToCache(ctx, f.ID)
	return nil
}

func (h *handler) pipecatcallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.pipecatcallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.pipecatcallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

func (h *handler) pipecatcallSetToCache(ctx context.Context, f *pipecatcall.Pipecatcall) error {
	if err := h.cache.PipecatcallSet(ctx, f); err != nil {
		return err
	}

	return nil
}

func (h *handler) pipecatcallGetFromCache(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {

	// get from cache
	res, err := h.cache.PipecatcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *handler) pipecatcallGetFromDB(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	query, args, err := squirrel.
		Select(pipecatcallsFields...).
		From(pipecatcallsTable).
		Where(squirrel.Eq{string(pipecatcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. pipecatcallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. pipecatcallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. pipecatcallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.pipecatcallGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. pipecatcallGetFromDB. id: %s", id)
	}

	return res, nil
}

func (h *handler) PipecatcallGet(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {

	res, err := h.pipecatcallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.pipecatcallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.pipecatcallSetToCache(ctx, res)

	return res, nil
}

func (h *handler) PipecatcallUpdate(ctx context.Context, id uuid.UUID, fields map[pipecatcall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[pipecatcall.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	return h.pipecatcallUpdate(ctx, id, fields)
}

func (h *handler) pipecatcallUpdate(ctx context.Context, id uuid.UUID, fields map[pipecatcall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	tmpFields := commondatabasehandler.PrepareUpdateFields(fields)
	q := squirrel.Update(pipecatcallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("pipecatcallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("pipecatcallUpdate: exec failed: %w", err)
	}

	_ = h.pipecatcallUpdateToCache(ctx, id)
	return nil
}

func (h *handler) PipecatcallDelete(ctx context.Context, id uuid.UUID) error {

	now := h.utilHandler.TimeGetCurTime()
	fields := map[pipecatcall.Field]any{
		pipecatcall.FieldTMDelete: now,
		pipecatcall.FieldTMUpdate: now,
	}

	if errUpdate := h.pipecatcallUpdate(ctx, id, fields); errUpdate != nil {
		return fmt.Errorf("could not update pipecatcall for delete. PipecatcallDelete. err: %v", errUpdate)
	}

	return nil
}
