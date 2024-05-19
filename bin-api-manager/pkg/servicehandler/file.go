package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CallCreate sends a request to call-manager
// to creating a call.
// it returns created calls and groupcalls info if it succeed.
func (h *serviceHandler) FileCreate(ctx context.Context, a *amagent.Agent, filepath string, name string, detail string) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FileCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new call.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, nil, fmt.Errorf("user has no permission")
	}

	h.reqHandler.Storage

	targetFlowID := flowID
	if targetFlowID == uuid.Nil {
		log.Debugf("The flowID is null. Creating a new temp flow for call dialing.")
		f, err := h.FlowCreate(ctx, a, "tmp", "tmp outbound flow", actions, false)
		if err != nil {
			log.Errorf("Could not create a flow for outoing call. err: %v", err)
			return nil, nil, err
		}
		log.WithField("flow", f).Debugf("Create a new tmp flow for call dialing. flow_id: %s", f.ID)

		targetFlowID = f.ID
	}

	// verify the flow
	f, err := h.flowGet(ctx, targetFlowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
		return nil, nil, err
	}
	if f.CustomerID != a.CustomerID {
		log.WithField("flow", f).Errorf("The flow has wrong customer id")
		return nil, nil, fmt.Errorf("the flow has wrong customer id")
	}

	tmpCalls, tmpGroupcalls, err := h.reqHandler.CallV1CallsCreate(ctx, a.CustomerID, targetFlowID, uuid.Nil, source, destinations, false, false)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, nil, err
	}

	resCalls := []*cmcall.WebhookMessage{}
	for _, tmp := range tmpCalls {
		t := tmp.ConvertWebhookMessage()
		resCalls = append(resCalls, t)
	}

	resGroupcalls := []*cmgroupcall.WebhookMessage{}
	for _, tmp := range tmpGroupcalls {
		t := tmp.ConvertWebhookMessage()
		resGroupcalls = append(resGroupcalls, t)
	}

	return resCalls, resGroupcalls, nil
}
