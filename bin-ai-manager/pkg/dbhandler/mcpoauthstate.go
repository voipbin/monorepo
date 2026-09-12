package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
)

const (
	mcpoauthstateTable = "ai_mcp_oauth_states"
)

// McpOAuthStateCreate creates a new short-lived OAuth state row (design
// §5, §7 step 2). TMCreate/TMExpire must already be set by the caller
// (mcpoauthhandler owns the 10-minute TTL policy, not the dbhandler).
func (h *handler) McpOAuthStateCreate(ctx context.Context, s *mcpoauthstate.McpOAuthState) error {
	fields, err := commondatabasehandler.PrepareFields(s)
	if err != nil {
		return fmt.Errorf("McpOAuthStateCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(mcpoauthstateTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("McpOAuthStateCreate: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("McpOAuthStateCreate: could not execute query. err: %v", err)
	}

	return nil
}

// McpOAuthStateGet returns the OAuth state row by its state token.
// Returns ErrNotFound if no row exists -- callers are responsible for
// additionally checking IsExpired() (design §7a Layer 1's TTL check is a
// policy decision, not a DB-layer concern).
func (h *handler) McpOAuthStateGet(ctx context.Context, state string) (*mcpoauthstate.McpOAuthState, error) {
	cols := commondatabasehandler.GetDBFields(mcpoauthstate.McpOAuthState{})

	query, args, err := sq.Select(cols...).
		From(mcpoauthstateTable).
		Where(sq.Eq{"state": state}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("McpOAuthStateGet: could not build query. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("McpOAuthStateGet: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &mcpoauthstate.McpOAuthState{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("McpOAuthStateGet: could not scan row. err: %v", err)
	}

	return res, nil
}

// McpOAuthStateDelete deletes the OAuth state row by its state token.
// Single-use per design §7a: /oauth/complete deletes the row immediately
// after the ownership check succeeds, BEFORE the token exchange -- never
// the public callback, which is a strictly read-only existence check.
func (h *handler) McpOAuthStateDelete(ctx context.Context, state string) error {
	query, args, err := sq.Delete(mcpoauthstateTable).
		Where(sq.Eq{"state": state}).
		ToSql()
	if err != nil {
		return fmt.Errorf("McpOAuthStateDelete: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("McpOAuthStateDelete: could not execute query. err: %v", err)
	}

	return nil
}

// McpOAuthStateDeleteExpired sweeps every OAuth state row past its
// tm_expire (design §5's periodic-sweep fallback for rows that were never
// consumed by /oauth/complete). Returns the number of rows deleted.
func (h *handler) McpOAuthStateDeleteExpired(ctx context.Context, now string) (int64, error) {
	query, args, err := sq.Delete(mcpoauthstateTable).
		Where(sq.Lt{"tm_expire": now}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("McpOAuthStateDeleteExpired: could not build query. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("McpOAuthStateDeleteExpired: could not execute query. err: %v", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("McpOAuthStateDeleteExpired: could not get rows affected. err: %v", err)
	}

	return n, nil
}
