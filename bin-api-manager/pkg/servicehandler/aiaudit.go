package servicehandler

import (
	"context"
	"regexp"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaiaudit "monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// bcp47Re validates BCP47 language tags.
var bcp47Re = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$`)

// AIAuditCreate triggers audit jobs for a completed AI call.
// It returns the list of created AIAudit records if it succeeds.
func (h *serviceHandler) AIAuditCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	aicallID uuid.UUID,
	language string,
) ([]*amaiaudit.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if language != "" && !bcp47Re.MatchString(language) {
		return nil, errors.Wrapf(serviceerrors.ErrInvalidArgument, "invalid BCP47 language tag: %s", language)
	}

	// Resolve the aicall to get the customer ID for permission check.
	aicall, err := h.aicallGet(ctx, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	if !h.hasPermission(ctx, a, aicall.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmps, err := h.reqHandler.AIV1AIAuditCreate(ctx, aicall.CustomerID, aicallID, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create ai audits")
	}

	res := make([]*amaiaudit.WebhookMessage, 0, len(tmps))
	for _, t := range tmps {
		res = append(res, t.ConvertWebhookMessage())
	}

	return res, nil
}

// aiauditGet returns the AIAudit record by ID without a permission check.
func (h *serviceHandler) aiauditGet(ctx context.Context, id uuid.UUID) (*amaiaudit.AIAudit, error) {
	res, err := h.reqHandler.AIV1AIAuditGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai audit info")
	}

	return res, nil
}

// AIAuditGetsByCustomerID returns a paginated list of AIAudit records for the authenticated customer.
func (h *serviceHandler) AIAuditGetsByCustomerID(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	aicallID uuid.UUID,
	aiID uuid.UUID,
) ([]*amaiaudit.WebhookMessage, error) {
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

	if aicallID != uuid.Nil {
		filters["aicall_id"] = aicallID.String()
	}

	if aiID != uuid.Nil {
		filters["ai_id"] = aiID.String()
	}

	typedFilters, err := h.convertAIAuditFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert ai audit filters")
	}

	tmps, err := h.reqHandler.AIV1AIAuditList(ctx, token, size, typedFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai audits info")
	}

	res := make([]*amaiaudit.WebhookMessage, 0, len(tmps))
	for _, t := range tmps {
		res = append(res, t.ConvertWebhookMessage())
	}

	return res, nil
}

// convertAIAuditFilters converts map[string]string to map[amaiaudit.Field]any.
func (h *serviceHandler) convertAIAuditFilters(filters map[string]string) (map[amaiaudit.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amaiaudit.AIAudit{})
	if err != nil {
		return nil, err
	}

	result := make(map[amaiaudit.Field]any, len(typed))
	for k, v := range typed {
		result[amaiaudit.Field(k)] = v
	}

	return result, nil
}

// AIAuditGet returns a single AIAudit by ID after checking ownership.
func (h *serviceHandler) AIAuditGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaiaudit.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.aiauditGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai audit info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return tmp.ConvertWebhookMessage(), nil
}

// AIAuditDelete soft-deletes an AIAudit record after checking ownership.
func (h *serviceHandler) AIAuditDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaiaudit.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.aiauditGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai audit info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1AIAuditDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete ai audit")
	}

	return res.ConvertWebhookMessage(), nil
}
