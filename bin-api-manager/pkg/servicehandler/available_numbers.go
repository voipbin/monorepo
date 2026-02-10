package servicehandler

import (
	"context"
	"fmt"

	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/sirupsen/logrus"
)

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) AvailableNumberList(ctx context.Context, a *amagent.Agent, size uint64, countryCode string, numType string) ([]*nmavailablenumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AvailableNumberGets",
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"size":         size,
		"country_code": countryCode,
		"type":         numType,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get available numbers
	filters := map[string]any{
		"customer_id":  a.CustomerID,
		"country_code": countryCode,
	}
	if numType != "" {
		filters["type"] = numType
	}
	tmps, err := h.reqHandler.NumberV1AvailableNumberList(ctx, size, filters)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*nmavailablenumber.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}
