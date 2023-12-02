package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
)

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) AvailableNumberGets(ctx context.Context, a *amagent.Agent, size uint64, countryCode string) ([]*nmavailablenumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AvailableNumberGets",
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"size":         size,
		"country_code": countryCode,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get available numbers
	tmps, err := h.reqHandler.NumberV1AvailableNumberGets(ctx, a.CustomerID, size, countryCode)
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
