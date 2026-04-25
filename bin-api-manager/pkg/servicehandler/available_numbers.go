package servicehandler

import (
	"context"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/sirupsen/logrus"
)

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) AvailableNumberList(ctx context.Context, a *auth.AuthIdentity, size uint64, countryCode string, numType string) ([]*nmavailablenumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AvailableNumberGets",
		"customer_id":  a.CustomerID,
		"username":     a.DisplayName(),
		"size":         size,
		"country_code": countryCode,
		"type":         numType,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
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
