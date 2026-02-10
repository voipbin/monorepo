package customerhandler

import (
	"context"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonbilling "monorepo/bin-common-handler/models/billing"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/pkg/metricshandler"
)

// IsValidBalance returns true if the customer's billing account has enough balance
func (h *customerHandler) IsValidBalance(ctx context.Context, customerID uuid.UUID, billingType bmbilling.ReferenceType, country string, count int) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "IsValidBalance",
		"customer_id":  customerID,
		"billing_type": billingType,
		"country":      country,
		"count":        count,
	})

	// get customer info
	c, err := h.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return false, errors.Wrap(err, "could not get customer info")
	}

	if count < 1 {
		count = 1
	}

	//
	start := time.Now()
	valid, err := h.reqHandler.BillingV1AccountIsValidBalance(ctx, c.BillingAccountID, billingType, country, count)
	elapsed := float64(time.Since(start).Milliseconds())
	metricshandler.RPCCallDuration.WithLabelValues("billing-manager", "is_valid_balance").Observe(elapsed)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		metricshandler.RPCCallTotal.WithLabelValues("billing-manager", "is_valid_balance", "error").Inc()
		return false, errors.Wrap(err, "could not get account info")
	}
	metricshandler.RPCCallTotal.WithLabelValues("billing-manager", "is_valid_balance", "success").Inc()

	return valid, nil
}

// IsValidResourceLimit returns true if the customer's billing account has not exceeded the resource limit for the given resource type.
func (h *customerHandler) IsValidResourceLimit(ctx context.Context, customerID uuid.UUID, resourceType commonbilling.ResourceType) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "IsValidResourceLimit",
		"customer_id":   customerID,
		"resource_type": resourceType,
	})

	c, err := h.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return false, errors.Wrap(err, "could not get customer info")
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	start := time.Now()
	valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimit(ctx, c.BillingAccountID, resourceType)
	elapsed := float64(time.Since(start).Milliseconds())
	metricshandler.RPCCallDuration.WithLabelValues("billing-manager", "is_valid_resource_limit").Observe(elapsed)
	if err != nil {
		log.Errorf("Could not validate the account's resource limit. err: %v", err)
		metricshandler.RPCCallTotal.WithLabelValues("billing-manager", "is_valid_resource_limit", "error").Inc()
		return false, errors.Wrap(err, "could not validate the account's resource limit")
	}
	metricshandler.RPCCallTotal.WithLabelValues("billing-manager", "is_valid_resource_limit", "success").Inc()

	return valid, nil
}
