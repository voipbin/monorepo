package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/team"
)

const (
	teamTable = "ai_teams"
)

// TeamCreate creates a new team record.
func (h *handler) TeamCreate(ctx context.Context, t *team.Team) error {
	t.TMCreate = h.utilHandler.TimeNow()
	t.TMUpdate = nil
	t.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("TeamCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(teamTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("TeamCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("TeamCreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.teamUpdateToCache(ctx, t.ID)

	return nil
}

// teamGetFromCache returns team from the cache.
func (h *handler) teamGetFromCache(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	res, err := h.cache.TeamGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// teamGetFromDB returns team from the DB.
func (h *handler) teamGetFromDB(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	cols := commondatabasehandler.GetDBFields(team.Team{})

	query, args, err := sq.Select(cols...).
		From(teamTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("teamGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("teamGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &team.Team{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("teamGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// teamUpdateToCache gets the team from the DB and updates the cache.
func (h *handler) teamUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.teamGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.teamSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// teamSetToCache sets the given team to the cache.
func (h *handler) teamSetToCache(ctx context.Context, t *team.Team) error {
	if err := h.cache.TeamSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// TeamGet returns team.
func (h *handler) TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	res, err := h.teamGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.teamGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.teamSetToCache(ctx, res)

	return res, nil
}

// TeamDelete deletes the team.
func (h *handler) TeamDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(teamTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("TeamDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("TeamDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.teamUpdateToCache(ctx, id)

	return nil
}

// TeamList returns a list of teams.
func (h *handler) TeamList(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(team.Team{})

	builder := sq.Select(cols...).
		From(teamTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("TeamList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("TeamList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*team.Team{}
	for rows.Next() {
		t := &team.Team{}
		if err := commondatabasehandler.ScanRow(rows, t); err != nil {
			return nil, fmt.Errorf("TeamList: could not scan row. err: %v", err)
		}
		res = append(res, t)
	}

	return res, nil
}

// TeamUpdate updates the team fields.
func (h *handler) TeamUpdate(ctx context.Context, id uuid.UUID, fields map[team.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("TeamUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(teamTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("TeamUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("TeamUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.teamUpdateToCache(ctx, id)

	return nil
}
