package servicehandler

import (
	"context"
	"fmt"
	"monorepo/bin-api-manager/models/auth"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

// ServiceAgentCustomerGet
// getting the defail of customer.
// it returns detail of customer if it succeed.
func (h *serviceHandler) ServiceAgentCustomerGet(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, fmt.Errorf("agent authentication required")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentCustomerGet",
		"agent": a,
	})

	tmp, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not customer info. err: %v", err)
		return nil, err
	}

	// copy the only limited information
	res := &cscustomer.WebhookMessage{
		ID: tmp.ID,

		Name:   tmp.Name,
		Detail: tmp.Detail,

		TMCreate: tmp.TMCreate,
		TMUpdate: tmp.TMUpdate,
		TMDelete: tmp.TMDelete,
	}

	return res, nil
}
