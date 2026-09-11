package aihandler

import (
	"context"
	stderrors "errors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// ValidateMcpServerIDs checks that every id in ids refers to an existing
// McpServer row owned by customerID. Returns a *cerrors.VoipbinError
// (InvalidArgument, surfaced as HTTP 400) naming the first non-existent or
// cross-customer id it finds -- listenhandler's errorResponse() only maps
// *cerrors.VoipbinError to a non-500 status, so a plain error here would
// otherwise reach the customer as an opaque 500 for a client-input mistake
// (mirrors the existing ai.ValidateToolNames -> cerrors.InvalidArgument
// pattern in chatbot.go).
//
// A genuine infra failure from McpServerGet (query build/exec/scan error,
// as opposed to dbhandler.ErrNotFound) is deliberately NOT mapped to 400
// here -- it is returned as-is so errorResponse() falls through to 500,
// matching the dbhandler.ErrNotFound-vs-other-error split ActivateInsight
// already established in db.go. Collapsing both into 400 would mislabel
// real DB outages as client mistakes and hide them from 5xx alerting.
//
// Lives in pkg/aihandler (has db access), not models/ai, mirroring why
// ai.ValidateToolNames itself takes no ctx/db today -- see
// docs/plans/2026-09-11-mcp-tool-integration-design.md §5.
func (h *aiHandler) ValidateMcpServerIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) error {
	for _, id := range ids {
		srv, err := h.db.McpServerGet(ctx, id)
		if err != nil {
			if stderrors.Is(err, dbhandler.ErrNotFound) {
				return cerrors.InvalidArgument(
					commonoutline.ServiceNameAIManager,
					"INVALID_MCP_SERVER_ID",
					"mcp_server_id "+id.String()+" is not accessible",
				).Wrap(err)
			}

			return errors.Wrapf(err, "could not get mcp server %s", id)
		}

		if srv == nil || srv.CustomerID != customerID {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameAIManager,
				"INVALID_MCP_SERVER_ID",
				"mcp_server_id "+id.String()+" is not accessible",
			)
		}
	}

	return nil
}

// UpdateMcpServerIDs persists the given McpServerIDs whitelist onto the AI
// identified by id and returns the updated AI. Callers (pkg/listenhandler)
// MUST call ValidateMcpServerIDs first with the AI's own customer_id to
// reject a cross-customer McpServer reference (IDOR) before calling this.
//
// This is a small, dedicated write path -- deliberately NOT a new
// positional parameter threaded through Create/Update -- to avoid the
// large mechanical rewrite of chatbot.go/chatbot_test.go's ~17 existing
// Create/Update call sites for a field that is independently validated and
// independently optional on every write.
func (h *aiHandler) UpdateMcpServerIDs(ctx context.Context, id uuid.UUID, mcpServerIDs []uuid.UUID) (*ai.AI, error) {
	fields := map[ai.Field]any{
		ai.FieldMcpServerIDs: mcpServerIDs,
	}

	if err := h.db.AIUpdate(ctx, id, fields); err != nil {
		return nil, errors.Wrapf(err, "could not update ai mcp_server_ids")
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated ai")
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)

	return res, nil
}
