package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// Asymmetric RBAC (design §7.1, review F6). Reads are low-privilege; trigger
// (paid LLM) and delete (mutating) require a higher floor so the lowest-
// privilege agent can never drive spend or destroy records.
const (
	// permTimelineAnalysisRead is the floor for GET/LIST: CustomerAgent+.
	permTimelineAnalysisRead = amagent.PermissionCustomerAgent |
		amagent.PermissionCustomerAdmin |
		amagent.PermissionCustomerManager
	// permTimelineAnalysisWrite is the floor for POST/DELETE: CustomerAdmin+.
	permTimelineAnalysisWrite = amagent.PermissionCustomerAdmin |
		amagent.PermissionCustomerManager
)

// TimelineAnalysisCreate triggers (or returns the existing/in-flight) AI
// analysis of an ended activeflow. The customer_id is server-injected from the
// authenticated token (the IDOR authority, review F2); the client never
// supplies it. Ownership and the ended-gate are enforced inside timeline-manager
// (foreign/absent activeflow -> masked not-found -> 404), so api-manager only
// forwards the authenticated customer_id.
func (h *serviceHandler) TimelineAnalysisCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	activeflowID uuid.UUID,
	reanalyze bool,
) (*tmanalysis.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if activeflowID == uuid.Nil {
		return nil, errors.Wrap(serviceerrors.ErrInvalidArgument, "activeflow_id is required")
	}

	if !h.hasPermission(ctx, a, a.CustomerID, permTimelineAnalysisWrite) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.TimelineV1AnalysisCreate(ctx, a.CustomerID, activeflowID, reanalyze)
	if err != nil {
		return nil, errors.Wrap(err, "could not create timeline analysis")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// TimelineAnalysisGet returns a single analysis after an ownership check. The
// customer_id is server-injected; timeline-manager masks a cross-customer or
// absent record as not-found (404), so no existence/ownership oracle leaks.
func (h *serviceHandler) TimelineAnalysisGet(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
) (*tmanalysis.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, permTimelineAnalysisRead) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.TimelineV1AnalysisGet(ctx, a.CustomerID, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get timeline analysis")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// TimelineAnalysisGetsByCustomerID returns a paginated list of the
// authenticated customer's analyses. customer_id is the server-injected
// authority filter (review F2); activeflow_id / status are optional client
// filters merged on top, but can never override customer_id.
func (h *serviceHandler) TimelineAnalysisGetsByCustomerID(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	activeflowID uuid.UUID,
	status tmanalysis.Status,
) ([]*tmanalysis.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, permTimelineAnalysisRead) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[tmanalysis.Field]any{
		tmanalysis.FieldDeleted: false,
	}
	if activeflowID != uuid.Nil {
		filters[tmanalysis.FieldActiveflowID] = activeflowID
	}
	if status != "" {
		filters[tmanalysis.FieldStatus] = string(status)
	}

	tmps, err := h.reqHandler.TimelineV1AnalysisList(ctx, a.CustomerID, token, size, filters)
	if err != nil {
		return nil, errors.Wrap(err, "could not list timeline analyses")
	}

	res := make([]*tmanalysis.WebhookMessage, 0, len(tmps))
	for i := range tmps {
		res = append(res, tmps[i].ConvertWebhookMessage())
	}

	return res, nil
}

// TimelineAnalysisDelete soft-deletes an analysis after an ownership check. The
// associated activeflow and its events are not affected. customer_id is
// server-injected; cross-customer/absent records are masked as not-found (404).
func (h *serviceHandler) TimelineAnalysisDelete(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
) (*tmanalysis.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, permTimelineAnalysisWrite) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.TimelineV1AnalysisDelete(ctx, a.CustomerID, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not delete timeline analysis")
	}

	return tmp.ConvertWebhookMessage(), nil
}
