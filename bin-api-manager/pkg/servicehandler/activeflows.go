package servicehandler

import (
	"context"
	"fmt"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// activeflowGet validates the activeflow's ownership and returns the activeflow info.
func (h *serviceHandler) activeflowGet(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "activeflowGet",
		"customer_id":   a.CustomerID,
		"agent_id":      a.CustomerID,
		"username":      a.Username,
		"activeflow_id": activeflowID,
	})

	// send request
	res, err := h.reqHandler.FlowV1ActiveflowGet(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get the activeflow info. err: %v", err)
		return nil, err
	}
	log.WithField("activeflow", res).Debug("Received result.")

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted activeflow.. activeflow_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// ActiveflowCreate sends a request to flow-manager
// to create a activeflow and execute.
// it returns created activeflow info if it succeed.
func (h *serviceHandler) ActiveflowCreate(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID, flowID uuid.UUID, actions []fmaction.Action) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowCreate",
		"agent":         a,
		"activeflow_id": activeflowID,
		"flow_id":       flowID,
		"actions":       actions,
	})
	log.Debug("Creating a new activeflow.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerManager|amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if activeflowID == uuid.Nil {
		activeflowID = h.utilHandler.UUIDCreate()
		log = log.WithField("activeflow_id", activeflowID)
		log.Debugf("The activeflow id is not valid. Generated a new activeflow id. activeflow_id: %s", activeflowID)
	}

	if flowID == uuid.Nil {
		log.Debugf("The flowID is null. Creating a new temp flow for call dialing.")
		tmpFlow, err := h.FlowCreate(ctx, a, "tmp", "tmp outbound flow", actions, false)
		if err != nil {
			log.Errorf("Could not create a flow for outoing call. err: %v", err)
			return nil, err
		}
		log.WithField("flow", tmpFlow).Debugf("Create a new tmp flow for call dialing. flow_id: %s", tmpFlow.ID)

		flowID = tmpFlow.ID
	}

	// verify the flow
	f, err := h.flowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
		return nil, err
	}
	if f.CustomerID != a.CustomerID {
		log.WithField("flow", f).Errorf("The flow has wrong customer id")
		return nil, fmt.Errorf("the flow has wrong customer id")
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, activeflowID, f.ID, fmactiveflow.ReferenceTypeNone, uuid.Nil)
	if err != nil {
		log.Errorf("Could not create activeflow. erR: %v", err)
		return nil, err
	}
	log.WithField("activeflow", af).Debugf("Created activeflow. activeflow_id: %s", af.ID)

	// execute created activeflow
	if err := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); err != nil {
		log.Errorf("Could not execute the activeflow. activeflow_id: %s", af.ID)
		return nil, err
	}

	res := af.ConvertWebhookMessage()
	return res, nil
}

// ActiveflowGet sends a request to flow-manager
// to getting a activeflow.
// it returns activeflow if it succeed.
func (h *serviceHandler) ActiveflowGet(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowGet",
		"customer_id":   a.CustomerID,
		"agent_id":      a.CustomerID,
		"activeflow_id": activeflowID,
	})

	// get activeflow
	tmp, err := h.activeflowGet(ctx, a, activeflowID)
	if err != nil {
		// no call info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// ActiveflowGets sends a request to flow-manager
// to getting a list of activeflows.
// it returns list of activeflows if it succeed.
func (h *serviceHandler) ActiveflowGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ActiveflowGets",
		"customer_id": a.CustomerID,
		"agent_id":    a.CustomerID,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	tmps, err := h.reqHandler.FlowV1ActiveflowGets(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get activeflows info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*fmactiveflow.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// ActiveflowStop sends a request to flow-manager
// to stopping the activeflow.
// it returns activeflow if it succeed.
func (h *serviceHandler) ActiveflowStop(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowStop",
		"customer_id":   a.CustomerID,
		"agent_id":      a.CustomerID,
		"activeflow_id": activeflowID,
	})

	// get activeflow
	af, err := h.activeflowGet(ctx, a, activeflowID)
	if err != nil {
		// no call info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1ActiveflowStop(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not stop the activeflow. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// ActiveflowDelete sends a request to flow-manager
// to delete the activeflow.
// it returns activeflow if it succeed.
func (h *serviceHandler) ActiveflowDelete(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowDelete",
		"customer_id":   a.CustomerID,
		"agent_id":      a.CustomerID,
		"activeflow_id": activeflowID,
	})

	af, err := h.activeflowGet(ctx, a, activeflowID)
	if err != nil {
		// no activeflow info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.FlowV1ActiveflowDelete(ctx, activeflowID)
	if err != nil {
		// no call info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()
	return res, nil
}
