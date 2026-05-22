package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaiprompthistory "monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIPromptHistoryGetsByAIID returns prompt history entries for the given AI.
// Permission check: caller must own the parent AI.
func (h *serviceHandler) AIPromptHistoryGetsByAIID(
	ctx context.Context,
	a *auth.AuthIdentity,
	aiID uuid.UUID,
	size uint64,
	token string,
) ([]*amaiprompthistory.AIPromptHistory, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	parentAI, err := h.aiGet(ctx, aiID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai info")
	}

	if !h.hasPermission(ctx, a, parentAI.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmps, err := h.reqHandler.AIV1AIPromptHistoryList(ctx, aiID, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt history list")
	}

	// Convert value slice to pointer slice.
	// AIPromptHistory has no WebhookMessage — all fields are appropriate for external
	// consumers; this is a justified deviation from the ConvertWebhookMessage() convention.
	res := make([]*amaiprompthistory.AIPromptHistory, len(tmps))
	for i := range tmps {
		res[i] = &tmps[i]
	}

	return res, nil
}

// AIPromptHistoryGet returns a single prompt history entry.
// Permission check: caller must own the parent AI.
func (h *serviceHandler) AIPromptHistoryGet(
	ctx context.Context,
	a *auth.AuthIdentity,
	aiID uuid.UUID,
	historyID uuid.UUID,
) (*amaiprompthistory.AIPromptHistory, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	parentAI, err := h.aiGet(ctx, aiID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai info")
	}

	if !h.hasPermission(ctx, a, parentAI.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1AIPromptHistoryGet(ctx, aiID, historyID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt history")
	}

	return res, nil
}
