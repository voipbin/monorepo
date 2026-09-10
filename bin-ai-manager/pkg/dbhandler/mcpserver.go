package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/mcpserver"
)

const (
	mcpserverTable = "ai_mcp_servers"
)

// McpServerCreate creates a new McpServer record.
//
// No cache layer for McpServer, unlike ai_ais (deliberate, see design doc
// §16): McpServer rows are read once per session (ListTools) plus once per
// tool call, bounded by conversation turn count rather than the AIcall-start
// hot path.
func (h *handler) McpServerCreate(ctx context.Context, m *mcpserver.McpServer) error {
	m.TMCreate = h.utilHandler.TimeNow()
	m.TMUpdate = nil
	m.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(m)
	if err != nil {
		return fmt.Errorf("McpServerCreate: could not prepare fields. err: %v", err)
	}
	delete(fields, "has_secret")

	query, args, err := sq.Insert(mcpserverTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("McpServerCreate: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("McpServerCreate: could not execute query. err: %v", err)
	}

	return nil
}

// mcpserverGetFromDB returns the McpServer from the DB. Mirrors
// dbhandler/team.go's TeamGet: does not filter tm_delete, so a caller that
// just soft-deleted a row (McpServerDelete followed by McpServerGet, the
// same pattern teamHandler.Delete uses) can still read back the row with
// tm_delete populated to confirm the delete and build the webhook payload.
func (h *handler) mcpserverGetFromDB(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error) {
	cols := commondatabasehandler.GetDBFields(mcpserver.McpServer{})

	query, args, err := sq.Select(cols...).
		From(mcpserverTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("mcpserverGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mcpserverGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &mcpserver.McpServer{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("mcpserverGetFromDB: could not scan row. err: %v", err)
	}
	res.HasSecret = len(res.SecretCiphertext) > 0

	return res, nil
}

// McpServerGet returns the McpServer by id.
func (h *handler) McpServerGet(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error) {
	return h.mcpserverGetFromDB(ctx, id)
}

// McpServerList returns a list of McpServers. Pass filters[mcpserver.FieldDeleted]=false
// (the ApplyFields/"deleted" convention shared with every other List in this
// package) to exclude soft-deleted rows.
func (h *handler) McpServerList(ctx context.Context, size uint64, token string, filters map[mcpserver.Field]any) ([]*mcpserver.McpServer, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(mcpserver.McpServer{})

	builder := sq.Select(cols...).
		From(mcpserverTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("McpServerList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("McpServerList: could not build query. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("McpServerList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*mcpserver.McpServer{}
	for rows.Next() {
		m := &mcpserver.McpServer{}
		if err := commondatabasehandler.ScanRow(rows, m); err != nil {
			return nil, fmt.Errorf("McpServerList: could not scan row. err: %v", err)
		}
		m.HasSecret = len(m.SecretCiphertext) > 0
		res = append(res, m)
	}

	return res, nil
}

// McpServerUpdate updates the McpServer fields.
func (h *handler) McpServerUpdate(ctx context.Context, id uuid.UUID, fields map[mcpserver.Field]any) error {
	updateFields := make(map[string]any, len(fields)+1)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("McpServerUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(mcpserverTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("McpServerUpdate: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("McpServerUpdate: could not execute. err: %v", err)
	}

	return nil
}

// McpServerDelete soft deletes the McpServer (tm_delete), mirroring
// Team/AI's soft-delete pattern.
func (h *handler) McpServerDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(mcpserverTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("McpServerDelete: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("McpServerDelete: could not execute. err: %v", err)
	}

	return nil
}
