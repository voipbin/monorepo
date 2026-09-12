package mcpoauthhandler

import (
	"context"
	stderrors "errors"

	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// CallbackExists implements McpOAuthHandler.CallbackExists (design §7a
// Layer 1, §10). It is a strictly read-only existence + expiry check --
// it never deletes the row, never touches McpServer, and never exchanges
// the authorization code. The public GET /mcpservers/oauth/callback
// handler is the only caller.
func (h *mcpOAuthHandler) CallbackExists(ctx context.Context, state string) (bool, error) {
	row, err := h.db.McpOAuthStateGet(ctx, state)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return false, nil
		}
		return false, errors.Wrap(err, "could not get oauth state")
	}

	now := h.utilHandler.TimeNow()
	if row.IsExpired(*now) {
		return false, nil
	}

	return true, nil
}
