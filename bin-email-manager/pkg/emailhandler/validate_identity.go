package emailhandler

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// validateCustomerIdentityVerified returns true if the given customer is allowed
// to send an email.
//
// It mirrors bin-call-manager's outgoing-PSTN identity gating, with one
// deliberate divergence: there is no destination-type branch. Email is an
// inherently external, billable send, so every send is gated. Internal/system
// traffic is covered by the internal-customer bypass, not by a type branch.
//
// Allow (true) when:
//   - the customer is an internal/system customer, OR
//   - the customer cannot be fetched (customer-manager unavailable -> fail-open), OR
//   - the customer fetch returns (nil, nil) (fail-open, parity with call-manager).
//
// Reject (false) when the customer is fetched (non-nil) and its identity
// verification status is not "verified".
//
// Fail-open rationale: availability with a bounded blast radius (the open window
// lasts only as long as the customer-manager outage). The fail-open branches log
// at WARN so an outage that widens the gate is visible.
func (h *emailHandler) validateCustomerIdentityVerified(ctx context.Context, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "validateCustomerIdentityVerified",
		"customer_id": customerID,
	})

	if cucustomer.IsInternalSystemID(customerID) {
		return true
	}

	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		// fail open: customer-manager unavailable.
		log.Warnf("Could not get customer info, failing open. customer_id: %s, err: %v", customerID, err)
		return true
	}
	if cu == nil {
		// fail open: (nil, nil) not-found shape, parity with call-manager.
		log.Warnf("Customer not found, failing open. customer_id: %s", customerID)
		return true
	}

	if cu.IdentityVerificationStatus != cucustomer.IdentityVerificationStatusVerified {
		log.Warnf("Customer identity not verified. Rejecting email send. customer_id: %s, status: %s", customerID, cu.IdentityVerificationStatus)
		return false
	}

	return true
}
