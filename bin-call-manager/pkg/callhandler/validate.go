package callhandler

import (
	"context"
	"strings"

	"monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/dongri/phonenumber"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
)

// ValidateCustomerNotFrozen returns the customer and true if the given customer is not frozen.
// Returns nil and false if the customer is frozen. Returns nil and true (fail-open) if
// customer-manager is unavailable.
func (h *callHandler) ValidateCustomerNotFrozen(ctx context.Context, customerID uuid.UUID) (*cucustomer.Customer, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerNotFrozen",
		"customer_id": customerID,
	})

	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		// Fail open: if customer-manager is unavailable, allow the call rather than
		// rejecting ALL calls. Billing-manager provides a second enforcement layer.
		log.Errorf("Could not get customer info, failing open. err: %v", err)
		return nil, true
	}
	log.WithField("customer", cu).Debugf("Retrieved customer info. customer_id: %s", cu.ID)

	if cu.Status == cucustomer.StatusFrozen {
		log.Infof("Customer account is frozen. Rejecting call.")
		return cu, false
	}

	return cu, true
}

// ValidateCustomerIdentityVerified returns true if the given customer has verified identity.
// Only checks for outgoing PSTN (TypeTel) calls. Inbound and non-PSTN calls skip this check.
// Known internal customer IDs bypass the check.
// Accepts a pre-fetched customer to avoid redundant RPC calls. If cu is nil (fail-open
// from frozen check), returns true.
func (h *callHandler) ValidateCustomerIdentityVerified(ctx context.Context, cu *cucustomer.Customer, customerID uuid.UUID, direction call.Direction, destination commonaddress.Address) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerIdentityVerified",
		"customer_id": customerID,
		"direction":   direction,
		"destination": destination,
	})

	// only check outgoing PSTN calls
	if direction != call.DirectionOutgoing || destination.Type != commonaddress.TypeTel {
		return true
	}

	// bypass for known internal/system customer IDs
	if customerID == cucustomer.IDCallManager ||
		customerID == cucustomer.IDAIManager ||
		customerID == cucustomer.IDSystem ||
		customerID == cucustomer.IDBasicRoute {
		log.Debugf("Internal customer ID, bypassing identity verification. customer_id: %s", customerID)
		return true
	}

	// if customer was not fetched (fail-open from frozen check), allow
	if cu == nil {
		log.Debugf("Customer not available (fail-open), bypassing identity verification.")
		return true
	}

	if cu.IdentityVerificationStatus != cucustomer.IdentityVerificationStatusVerified {
		log.Infof("Customer identity not verified. Rejecting outgoing PSTN call. customer_id: %s, status: %s", customerID, cu.IdentityVerificationStatus)
		return false
	}

	return true
}

// ValidateCustomerBalance returns true if the given customer has enough balance
func (h *callHandler) ValidateCustomerBalance(
	ctx context.Context,
	callID uuid.UUID,
	customerID uuid.UUID,
	direction call.Direction,
	source commonaddress.Address,
	destination commonaddress.Address,
) bool {
	log := logrus.WithFields(logrus.Fields{
		"funcs":       "ValidateCustomerBalance",
		"call_id":     callID,
		"customer_id": customerID,
		"direction":   direction,
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Validating the customer's balance. call_id: %s", callID)

	// get country
	var country string
	if direction == call.DirectionIncoming {
		country = h.getCountry(ctx, source.Target)
	} else {
		country = h.getCountry(ctx, destination.Target)
	}
	validBalance, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, billing.ReferenceTypeCall, country, 1)

	if err != nil {
		log.Errorf("Could not check the account balance. err: %v", err)
		return false
	}

	if !validBalance {
		log.Infof("The account has not enough balance.")
		return false
	}

	return true
}

// ValidateDestination returns true if the given customer has enough balance
func (h *callHandler) ValidateDestination(ctx context.Context, customerID uuid.UUID, destination commonaddress.Address) bool {
	log := logrus.WithFields(logrus.Fields{
		"funcs":       "ValidateDestination",
		"customer_id": customerID,
		"destination": destination,
	})

	// todo: need to implement
	log.Debug("Pass the destination validation.")
	return true
}

// ValidateDestination returns true if the given customer has enough balance
func (h *callHandler) getCountry(ctx context.Context, number string) string {
	n := phonenumber.Parse(number, "")
	country := phonenumber.GetISO3166ByNumber(n, false)

	res := strings.ToLower(country.Alpha2)
	return res
}
