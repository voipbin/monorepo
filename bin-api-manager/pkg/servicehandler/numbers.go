package servicehandler

import (
	"context"
	"fmt"

	nmnumber "monorepo/bin-number-manager/models/number"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// numberGet validates the number's ownership and returns the number info.
func (h *serviceHandler) numberGet(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "numberGet",
		"number_id": id,
	})

	// get number info
	res, err := h.reqHandler.NumberV1NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	if res.TMDelete != defaultTimestamp {
		log.WithField("number", res).Debugf("Deleted number.")
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// NumberGets sends a request to getting a list of numbers
// It sends a request to the number-manager to getting a list of numbers.
// it returns list of numbers if it succeed.
func (h *serviceHandler) NumberGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberGets",
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

	// get available numbers
	// Convert string filters to typed filters
	typedFilters, err := h.convertNumberFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.NumberV1NumberGets(ctx, token, size, typedFilters)
	if err != nil {
		log.Infof("Could not get numbers info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*nmnumber.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// NumberCreate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberCreate(ctx context.Context, a *amagent.Agent, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"numbers":     num,
	})

	if num == "" {
		log.Errorf("Not acceptable number. num: %s", num)
		return nil, fmt.Errorf("not acceptable number. num: %s", num)
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create numbers
	tmp, err := h.reqHandler.NumberV1NumberCreate(ctx, a.CustomerID, num, callFlowID, messageFlowID, name, detail)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberGet handles number get request.
// It sends a request to the number-manager to get a existed number.
// it returns got number information if it succeed.
func (h *serviceHandler) NumberGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"number":      id,
	})

	// get number info
	tmp, err := h.numberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberDelete handles number delete request.
// It sends a request to the number-manager to delete a existed number.
// it returns deleted number information if it succeed.
func (h *serviceHandler) NumberDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"number":      id,
	})

	// get number info
	n, err := h.numberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, n.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// delete numbers
	tmp, err := h.reqHandler.NumberV1NumberDelete(ctx, id)
	if err != nil {
		log.Infof("Could not delete numbers info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberUpdate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "NumberUpdate",
		"agent":           a,
		"number_id":       id,
		"call_flow_id":    callFlowID,
		"message_flow_id": messageFlowID,
	})

	// get number
	n, err := h.numberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, n.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// check call flow
	if callFlowID != uuid.Nil && !h.numberVerifyFlow(ctx, a, callFlowID) {
		log.Errorf("Could not verify call flow")
		return nil, fmt.Errorf("could not verify call flow")
	}

	// check message flow
	if messageFlowID != uuid.Nil && !h.numberVerifyFlow(ctx, a, messageFlowID) {
		log.Errorf("Could not verify message flow")
		return nil, fmt.Errorf("could not verify message flow")
	}

	// update number
	tmp, err := h.reqHandler.NumberV1NumberUpdate(ctx, id, callFlowID, messageFlowID, name, detail)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberUpdate handles number create request.
// It sends a request to the number-manager to create a new number.
// it returns created number information if it succeed.
func (h *serviceHandler) NumberUpdateFlowIDs(ctx context.Context, a *amagent.Agent, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberUpdateFlowIDs",
		"customer_id": a.CustomerID,
		"number_id":   id,
	})

	// get number
	n, err := h.numberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, n.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// update number
	tmp, err := h.reqHandler.NumberV1NumberUpdateFlowID(ctx, id, callFlowID, messageFlowID)
	if err != nil {
		log.Errorf("Could not update the number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// NumberRenew handles number renew request.
// It sends a request to the number-manager to renew the numbers.
// it returns renewed numbers information if it succeed.
func (h *serviceHandler) NumberRenew(ctx context.Context, a *amagent.Agent, tmRenew string) ([]*nmnumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "NumberRenew",
		"customer_id": a.CustomerID,
		"tm_renew":    tmRenew,
	})

	// project admin only
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// get number
	tmps, err := h.reqHandler.NumberV1NumberRenewByTmRenew(ctx, tmRenew)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, err
	}

	res := []*nmnumber.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// numberVerifyFlow returns true if the agent has a correct permission for the given flow
func (h *serviceHandler) numberVerifyFlow(ctx context.Context, a *amagent.Agent, flowID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":    "numberVerifyFlow",
		"agent":   a,
		"flow_id": flowID,
	})

	if flowID == uuid.Nil {
		return true
	}

	f, err := h.flowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
		return false
	}

	if f.CustomerID != a.CustomerID {
		log.Errorf("The flow has different customer id.")
		return false
	}

	return true
}

// convertNumberFilters converts map[string]string to map[nmnumber.Field]any
func (h *serviceHandler) convertNumberFilters(filters map[string]string) (map[nmnumber.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, nmnumber.Number{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[nmnumber.Field]any, len(typed))
	for k, v := range typed {
		result[nmnumber.Field(k)] = v
	}

	return result, nil
}
