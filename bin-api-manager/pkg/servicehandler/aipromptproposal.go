package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIPromptProposalCreate triggers a prompt improvement proposal for an AI based on selected audits.
// It returns the created AIPromptProposal record if it succeeds.
func (h *serviceHandler) AIPromptProposalCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	aiID uuid.UUID,
	auditIDs []uuid.UUID,
	language string,
) (*amaipromptproposal.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Resolve the AI to get the customer ID for permission check.
	parentAI, err := h.aiGet(ctx, aiID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai info")
	}

	if !h.hasPermission(ctx, a, parentAI.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.AIV1AIPromptProposalCreate(ctx, parentAI.CustomerID, aiID, auditIDs, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create ai prompt proposal")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// aipromptproposalGet returns the AIPromptProposal record by ID without a permission check.
func (h *serviceHandler) aipromptproposalGet(ctx context.Context, id uuid.UUID) (*amaipromptproposal.AIPromptProposal, error) {
	res, err := h.reqHandler.AIV1AIPromptProposalGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt proposal info")
	}

	return res, nil
}

// AIPromptProposalGetsByCustomerID returns a paginated list of AIPromptProposal records for the authenticated customer.
func (h *serviceHandler) AIPromptProposalGetsByCustomerID(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	aiID uuid.UUID,
	status amaipromptproposal.Status,
) ([]*amaipromptproposal.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"deleted":     "false",
		"customer_id": a.CustomerID.String(),
	}

	if aiID != uuid.Nil {
		filters["ai_id"] = aiID.String()
	}

	if status != "" {
		filters["status"] = string(status)
	}

	typedFilters, err := h.convertAIPromptProposalFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert ai prompt proposal filters")
	}

	tmps, err := h.reqHandler.AIV1AIPromptProposalList(ctx, token, size, typedFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt proposals info")
	}

	res := make([]*amaipromptproposal.WebhookMessage, 0, len(tmps))
	for _, t := range tmps {
		res = append(res, t.ConvertWebhookMessage())
	}

	return res, nil
}

// convertAIPromptProposalFilters converts map[string]string to map[amaipromptproposal.Field]any.
func (h *serviceHandler) convertAIPromptProposalFilters(filters map[string]string) (map[amaipromptproposal.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amaipromptproposal.AIPromptProposal{})
	if err != nil {
		return nil, err
	}

	result := make(map[amaipromptproposal.Field]any, len(typed))
	for k, v := range typed {
		result[amaipromptproposal.Field(k)] = v
	}

	return result, nil
}

// AIPromptProposalGet returns a single AIPromptProposal by ID after checking ownership.
func (h *serviceHandler) AIPromptProposalGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.aipromptproposalGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt proposal info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return tmp.ConvertWebhookMessage(), nil
}

// AIPromptProposalAccept accepts a proposal and applies it to the AI's prompt.
func (h *serviceHandler) AIPromptProposalAccept(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.aipromptproposalGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt proposal info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1AIPromptProposalAccept(ctx, tmp.CustomerID, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not accept ai prompt proposal")
	}

	return res.ConvertWebhookMessage(), nil
}

// AIPromptProposalReject rejects a proposal.
func (h *serviceHandler) AIPromptProposalReject(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.aipromptproposalGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt proposal info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1AIPromptProposalReject(ctx, tmp.CustomerID, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not reject ai prompt proposal")
	}

	return res.ConvertWebhookMessage(), nil
}

// AIPromptProposalDelete soft-deletes an AIPromptProposal record after checking ownership.
func (h *serviceHandler) AIPromptProposalDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.aipromptproposalGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai prompt proposal info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1AIPromptProposalDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete ai prompt proposal")
	}

	return res.ConvertWebhookMessage(), nil
}
