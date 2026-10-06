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
// McpServer row owned by customerID. An id that is NOT already in storedIDs
// must additionally be not-deleted. Returns a
// *cerrors.VoipbinError (InvalidArgument, surfaced as HTTP 400) naming the
// first inaccessible id it finds -- listenhandler's errorResponse() only maps
// *cerrors.VoipbinError to a non-500 status, so a plain error here would
// otherwise reach the customer as an opaque 500 for a client-input mistake
// (mirrors the existing ai.ValidateToolNames -> cerrors.InvalidArgument
// pattern in chatbot.go).
//
// "Not deleted" is checked here and not left to McpServerGet, which returns
// soft-deleted rows on purpose (the REST read of a deleted server answers
// 200). Without the TMDelete check a customer could whitelist a server they
// had already deleted, and the AI would carry an id that no consumer will
// ever honour.
//
// storedIDs is that check's ONLY exemption, and it exists to keep an AI
// saveable (D21). Deleting an MCP server does not prune the id from any AI
// that referenced it, and square-admin's picker is built from the
// deleted:"false" list, so the stale id is invisible and unremovable in the
// UI while every PUT re-submits it. Rejecting it would 400 every save of
// that AI forever, including edits to unrelated fields like name or prompt.
// So a deleted id the AI already carries is tolerated; a deleted id the
// request is ADDING is still rejected.
//
// Existence and ownership are NOT exempted for stored ids (D25): a stored id
// whose row is gone, or whose row now belongs to another customer, is still
// rejected. The exemption is narrow on purpose -- widening it to skip stored
// ids entirely would turn the whitelist into a place where a cross-customer
// reference, once stored, could never be caught again.
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
func (h *aiHandler) ValidateMcpServerIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID, storedIDs []uuid.UUID) error {
	stored := make(map[uuid.UUID]struct{}, len(storedIDs))
	for _, id := range storedIDs {
		stored[id] = struct{}{}
	}

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

		// Deleted rows: rejected unless the AI already carries this id.
		if _, alreadyStored := stored[id]; !alreadyStored && srv.TMDelete != nil {
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

	h.publishAIEvent(ctx, ai.EventTypeUpdated, res)

	return res, nil
}
