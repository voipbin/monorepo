package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ServiceAgentAImessageList sends a request to ai-manager to get a list of
// aimessages for the given aicall id. The aicall must belong to the service
// agent's own customer (tenant isolation only, no ownership check).
func (h *serviceHandler) ServiceAgentAImessageList(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, size uint64, token string) ([]*ammessage.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentAImessageList",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"aicall_id":   aicallID,
		"size":        size,
		"token":       token,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// confirm the aicall belongs to the agent's own tenant before listing
	// its messages -- tenant isolation only, no ownership check.
	if _, err := h.ServiceAgentAIcallGet(ctx, a, aicallID); err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. aicall_id: %v", aicallID)
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if size == 0 {
		size = 100
	}

	filters := map[string]string{
		"deleted":   "false",
		"aicall_id": aicallID.String(),
	}

	typedFilters, err := h.convertAImessageFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert filters")
	}

	tmps, err := h.reqHandler.AIV1MessageGetsByAIcallID(ctx, aicallID, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get aimessages. err: %v", err)
		return nil, errors.Wrapf(err, "could not get aimessages info")
	}

	res := []*ammessage.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// ServiceAgentAImessageCreate sends a request to ai-manager to send a new
// message to the given aicall. The aicall must belong to the service agent's
// own customer (tenant isolation only, no ownership check).
func (h *serviceHandler) ServiceAgentAImessageCreate(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, role ammessage.Role, content string) (*ammessage.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentAImessageCreate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"aicall_id":   aicallID,
		"role":        role,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// confirm the aicall belongs to the agent's own tenant before sending a
	// message to it -- tenant isolation only, no ownership check.
	if _, err := h.ServiceAgentAIcallGet(ctx, a, aicallID); err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. aicall_id: %v", aicallID)
	}

	tmp, err := h.reqHandler.AIV1MessageSend(ctx, aicallID, role, content, true, false, 30000)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new ai message. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentAImessageDraftDecision sends a request to ai-manager to approve
// or reject an SMS draft card of the given aicall. The aicall must belong to
// the service agent's own customer (tenant isolation only, no ownership
// check): any agent of the customer can decide any draft of the customer,
// the same permission as the other aimessage calls. The draft itself (its
// recipient, text, owner and age) is validated by ai-manager.
func (h *serviceHandler) ServiceAgentAImessageDraftDecision(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, draftID uuid.UUID, decision ammessage.DraftDecision) (*ammessage.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentAImessageDraftDecision",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"aicall_id":   aicallID,
		"draft_id":    draftID,
		"decision":    decision,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// confirm the aicall belongs to the agent's own tenant before deciding
	// a draft of it -- tenant isolation only, no ownership check.
	if _, err := h.ServiceAgentAIcallGet(ctx, a, aicallID); err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. aicall_id: %v", aicallID)
	}

	tmp, err := h.reqHandler.AIV1MessageDraftDecision(ctx, aicallID, draftID, decision)
	if err != nil {
		return nil, errors.Wrapf(err, "could not decide the draft. err: %v", err)
	}

	// the deciding agent is recorded in the log only.
	log.Infof("Decided the sms draft. result_message_id: %s", tmp.ID)

	return tmp.ConvertWebhookMessage(), nil
}
