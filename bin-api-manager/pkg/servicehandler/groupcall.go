package servicehandler

import (
	"context"
	"fmt"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// groupcallGet validates the call's ownership and returns the call info.
func (h *serviceHandler) groupcallGet(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "groupcallGet",
		"groupcall_id": groupcallID,
	})

	// send request
	res, err := h.reqHandler.CallV1GroupcallGet(ctx, groupcallID)
	if err != nil {
		log.Errorf("Could not get the groupcall info. err: %v", err)
		return nil, err
	}
	log.WithField("groupcall", res).Debug("Received result.")

	return res, nil
}

// CallGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) GroupcallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GroupcallGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	// get calls
	// Convert string filters to typed filters
	typedFilters, err := h.convertGroupcallFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.CallV1GroupcallList(ctx, token, size, typedFilters)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*cmgroupcall.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// GroupcallGet sends a request to call-manager
// to getting a groupcall.
// it returns groupcall if it succeed.
func (h *serviceHandler) GroupcallGet(ctx context.Context, a *amagent.Agent, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GroupcallGet",
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"groupcall_id": groupcallID,
	})

	// get call
	c, err := h.groupcallGet(ctx, groupcallID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// convert
	res := c.ConvertWebhookMessage()
	return res, nil
}

// GroupcallCreate sends a request to call-manager
// to creating a groupcall.
// it returns created groupcall info if it succeed.
func (h *serviceHandler) GroupcallCreate(ctx context.Context, a *amagent.Agent, source commonaddress.Address, destinations []commonaddress.Address, flowID uuid.UUID, actions []fmaction.Action, ringMethod cmgroupcall.RingMethod, answerMethod cmgroupcall.AnswerMethod) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "GroupcallCreate",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"flow_id":       flowID,
		"actions":       actions,
		"source":        source,
		"destinations":  destinations,
		"ring_method":   ringMethod,
		"answer_method": answerMethod,
	})
	log.Debug("Creating a new groupcall.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	targetFlowID := flowID
	if targetFlowID == uuid.Nil {
		log.Debugf("The flowID is null. Creating a new temp flow for call dialing.")
		f, err := h.FlowCreate(ctx, a, "tmp", "tmp outbound flow", actions, uuid.Nil, false)
		if err != nil {
			log.Errorf("Could not create a flow for outoing call. err: %v", err)
			return nil, err
		}
		log.WithField("flow", f).Debugf("Create a new tmp flow for call dialing. flow_id: %s", f.ID)

		targetFlowID = f.ID
	}

	tmp, err := h.reqHandler.CallV1GroupcallCreate(ctx, uuid.Nil, a.CustomerID, targetFlowID, source, destinations, uuid.Nil, uuid.Nil, ringMethod, answerMethod)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, err
}

// GroupcallHangup sends a request to groupcall-manager
// to hangup the groupcall.
// it returns groupcall if it succeed.
func (h *serviceHandler) GroupcallHangup(ctx context.Context, a *amagent.Agent, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GroupcallHangup",
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"groupcall_id": groupcallID,
	})

	gc, err := h.groupcallGet(ctx, groupcallID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, gc.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.CallV1GroupcallHangup(ctx, groupcallID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// GroupcallDelete sends a request to groupcall-manager
// to delete the groupcall.
// it returns groupcall if it succeed.
func (h *serviceHandler) GroupcallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GroupcallDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"call_id":     callID,
	})

	gc, err := h.groupcallGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, gc.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.CallV1GroupcallDelete(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// convertGroupcallFilters converts map[string]string to map[cmgroupcall.Field]any
func (h *serviceHandler) convertGroupcallFilters(filters map[string]string) (map[cmgroupcall.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, cmgroupcall.Groupcall{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[cmgroupcall.Field]any, len(typed))
	for k, v := range typed {
		result[cmgroupcall.Field(k)] = v
	}

	return result, nil
}
