package servicehandler

import (
	"context"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

// ServiceAgentCustomerGet
// getting the defail of customer.
// it returns detail of customer if it succeed.
func (h *serviceHandler) ServiceAgentCustomerGet(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
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
