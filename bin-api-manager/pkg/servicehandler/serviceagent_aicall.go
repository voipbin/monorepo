package servicehandler

import (
	"context"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ServiceAgentAIcallList sends a request to ai-manager to get a list of
// aicalls for the service agent's customer.
// it returns list of aicall info if it succeed.
// If referenceType/referenceID are both provided, results are additionally
// filtered to aicalls originating from that specific resource (e.g. a
// contact case). This lets a service agent frontend (e.g. square-talk) check
// whether an aicall is already in progress for a given reference before
// starting a new one. status can additionally be supplied (independently of
// referenceType/referenceID) to narrow to a specific lifecycle state (e.g.
// "progressing") so a prior, already-terminated aicall for the same
// reference is not mistaken for one still in progress.
// The caller (server/service_agents_aicalls.go) is expected to reject a
// partial pair (only one of referenceType/referenceID non-zero) before
// calling this; this function does not itself validate pairing.
func (h *serviceHandler) ServiceAgentAIcallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, referenceType string, referenceID uuid.UUID, status string) ([]*amaicall.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceAgentAIcallList",
		"customer_id":    a.CustomerID,
		"username":       a.DisplayName(),
		"size":           size,
		"token":          token,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"status":         status,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}
	if referenceType != "" {
		filters["reference_type"] = referenceType
	}
	if referenceID != uuid.Nil {
		filters["reference_id"] = referenceID.String()
	}
	if status != "" {
		filters["status"] = status
	}

	typedFilters, err := h.convertAIcallFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert filters")
	}

	tmps, err := h.reqHandler.AIV1AIcallList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get aicalls. err: %v", err)
		return nil, errors.Wrapf(err, "could not get aicalls info")
	}

	res := []*amaicall.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// ServiceAgentAIcallGet sends a request to ai-manager to get the aicall info
// for the service agent's customer. it returns the aicall info if it succeed.
// Tenant isolation only -- no ownership check beyond the customer match.
func (h *serviceHandler) ServiceAgentAIcallGet(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID) (*amaicall.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentAIcallGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"aicall_id":   aicallID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.aicallGet(ctx, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. aicall_id: %v", aicallID)
	}

	if tmp.CustomerID != a.CustomerID {
		log.Info("The aicall does not belong to the agent's customer.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentAIcallCreate sends a request to ai-manager to create an aicall
// for the service agent's customer. An activeflow is automatically created
// and associated with the new aicall, mirroring the top-level AIcallCreate.
// it returns the created aicall info if it succeed.
func (h *serviceHandler) ServiceAgentAIcallCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	assistanceType amaicall.AssistanceType,
	assistanceID uuid.UUID,
	referenceType amaicall.ReferenceType,
	referenceID uuid.UUID,
) (*amaicall.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":            "ServiceAgentAIcallCreate",
		"customer_id":     a.CustomerID,
		"assistance_type": assistanceType,
		"assistance_id":   assistanceID,
		"reference_type":  referenceType,
		"reference_id":    referenceID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Case-ownership check (contact_case only). Rejects before touching
	// anything else if the referenced Case doesn't belong to the caller's
	// own customer -- fail-fast; tool_insight.go already fails closed on
	// this independently, so this is a UX/hygiene guard, not the last line
	// of defense.
	if referenceType == amaicall.ReferenceTypeContactCase {
		kase, errCase := h.reqHandler.ContactV1CaseGet(ctx, a.CustomerID, referenceID)
		if errCase != nil || kase.CustomerID != a.CustomerID {
			log.Info("The referenced case does not belong to the agent's customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}

	// Server-side Insight AI resolution: assistance_id may be omitted for
	// assistance_type=ai + reference_type=contact_case. This is the only
	// combination the schema allows it to be omitted for.
	if assistanceType == amaicall.AssistanceTypeAI && assistanceID == uuid.Nil {
		if referenceType != amaicall.ReferenceTypeContactCase {
			return nil, errors.Wrapf(serviceerrors.ErrInvalidArgument, "assistance_id is required for reference_type: %s", referenceType)
		}

		resolvedID, errResolve := h.resolveInsightAIID(ctx, a.CustomerID)
		if errResolve != nil {
			return nil, errResolve
		}
		assistanceID = resolvedID
		log.WithField("assistance_id", assistanceID).Debugf("Resolved the customer's insight AI.")
	}

	// resolve the assistance entity's customer id and confirm it belongs to
	// the agent's own tenant. this is tenant isolation only -- no ownership
	// check on the assistance entity itself.
	var customerID uuid.UUID
	switch assistanceType {
	case amaicall.AssistanceTypeAI:
		cb, err := h.aiGet(ctx, assistanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get ai info")
		}
		customerID = cb.CustomerID
	case amaicall.AssistanceTypeTeam:
		t, err := h.teamGet(ctx, assistanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get team info")
		}
		customerID = t.CustomerID
	default:
		return nil, errors.Wrapf(serviceerrors.ErrInvalidArgument, "unsupported assistance type: %s", assistanceType)
	}

	if customerID != a.CustomerID {
		log.Info("The assistance entity does not belong to the agent's customer.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// create activeflow for the aicall
	af, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		a.CustomerID,
		uuid.Nil,
		fmactiveflow.ReferenceTypeAPI,
		uuid.Nil,
		uuid.Nil,
		nil,
		"",
		fmactiveflow.WebhookMethodNone,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create activeflow for aicall")
	}
	log.WithField("activeflow", af).Debugf("Created activeflow for aicall. activeflow_id: %s", af.ID)

	tmp, err := h.reqHandler.AIV1AIcallStart(
		ctx,
		assistanceType,
		assistanceID,
		af.ID,
		referenceType,
		referenceID,
	)
	if err != nil {
		// best-effort cleanup of the orphaned activeflow
		if _, errDelete := h.reqHandler.FlowV1ActiveflowDelete(ctx, af.ID); errDelete != nil {
			log.Errorf("Could not delete orphaned activeflow. activeflow_id: %s, err: %v", af.ID, errDelete)
		}
		return nil, errors.Wrapf(err, "could not create aicall")
	}

	// If AIV1AIcallStart returned a REUSED aicall (its ActiveflowID differs
	// from the activeflow this call just created above), the activeflow we
	// created is now orphaned -- clean it up the same way the error path
	// above does. Otherwise ServiceAgentAIcallCreate leaks one activeflow
	// per call against an already-live contact_case session.
	if tmp.ActiveflowID != af.ID {
		if _, errDelete := h.reqHandler.FlowV1ActiveflowDelete(ctx, af.ID); errDelete != nil {
			log.Errorf("Could not delete orphaned activeflow after aicall reuse. activeflow_id: %s, err: %v", af.ID, errDelete)
		}
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// resolveInsightAIID looks up the caller's own customer's single
// type=insight AI. Returns serviceerrors.ErrNotFound if none exists (the
// square-admin panel maps this to its empty state), or the
// most-recently-created one if 2+ exist (AIV1AIList orders tm_create desc).
func (h *serviceHandler) resolveInsightAIID(ctx context.Context, customerID uuid.UUID) (uuid.UUID, error) {
	filters, err := h.convertAIFilters(map[string]string{
		"deleted":     "false",
		"customer_id": customerID.String(),
		"type":        string(amai.TypeInsight),
	})
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "could not convert ai filters")
	}

	ais, err := h.reqHandler.AIV1AIList(ctx, "", 100, filters)
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "could not list ais")
	}
	if len(ais) == 0 {
		return uuid.Nil, serviceerrors.ErrNotFound
	}

	return ais[0].ID, nil
}
