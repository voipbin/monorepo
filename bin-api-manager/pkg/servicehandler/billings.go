package servicehandler

import (
	"context"
	"fmt"

	bmbilling "monorepo/bin-billing-manager/models/billing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/sirupsen/logrus"
)

// BillingGets sends a request to billing-manager
// to getting a list of billings.
// it returns list of billings if it succeed.
func (h *serviceHandler) BillingGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*bmbilling.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
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

	// Convert string filters to typed filters
	typedFilters, err := h.convertBillingFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	// get billings
	tmps, err := h.reqHandler.BillingV1BillingGets(ctx, token, size, typedFilters)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*bmbilling.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// convertBillingFilters converts map[string]string to map[bmbilling.Field]any
func (h *serviceHandler) convertBillingFilters(filters map[string]string) (map[bmbilling.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, bmbilling.Billing{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[bmbilling.Field]any, len(typed))
	for k, v := range typed {
		result[bmbilling.Field(k)] = v
	}

	return result, nil
}
