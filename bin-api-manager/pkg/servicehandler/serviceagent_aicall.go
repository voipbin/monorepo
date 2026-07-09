package servicehandler

import (
	"context"

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

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
