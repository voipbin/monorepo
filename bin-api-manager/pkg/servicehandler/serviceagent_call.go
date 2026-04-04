package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentCallGets sends a request to call-manager
// to getting the given agent's list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) ServiceAgentCallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmcall.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, fmt.Errorf("agent authentication required")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCallGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// filters
	filters := map[string]string{
		"owner_id":    a.AgentID().String(),
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	res, err := h.callGetsByFilters(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentCallGet sends a request to call-manager
// to getting a call.
// it returns call if it succeed.
func (h *serviceHandler) ServiceAgentCallGet(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, fmt.Errorf("agent authentication required")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":    "ServiceAgentCallGet",
		"agent":   a,
		"call_id": callID,
	})

	// get call
	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if a.AgentID() != c.OwnerID {
		return nil, fmt.Errorf("user has no permission")
	}

	// convert
	res := c.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentCallDelete sends a request to call-manager
// to delete the call.
// it returns deleted call if it succeed.
func (h *serviceHandler) ServiceAgentCallDelete(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, fmt.Errorf("agent authentication required")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":    "ServiceAgentCallDelete",
		"agent":   a,
		"call_id": callID,
	})

	// get call
	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if a.AgentID() != c.OwnerID {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.callDelete(ctx, callID)
	if err != nil {
		log.Errorf("Could not delete call info. err: %v", err)
		return nil, err
	}

	return res, nil
}
