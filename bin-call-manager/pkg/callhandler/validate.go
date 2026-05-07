package callhandler

import (
	"context"
	"fmt"
	"strings"

	"monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/dongri/phonenumber"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

// ValidateCustomerStatusOutgoing returns the customer and true if the customer status is active.
// Only active customers are allowed to make outgoing calls.
// Returns (customer, false) if the status is not active.
// Returns (nil, true) if customer-manager is unavailable (fail-open).
func (h *callHandler) ValidateCustomerStatusOutgoing(ctx context.Context, customerID uuid.UUID) (*cucustomer.Customer, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerStatusOutgoing",
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

	if cu.Status != cucustomer.StatusActive {
		log.Infof("Customer account is not active. Rejecting outgoing call. status: %s", cu.Status)
		return cu, false
	}

	return cu, true
}

// ValidateCustomerStatusIncoming returns the customer and true if the customer status is active or initial.
// Active and initial customers are allowed to receive incoming calls.
// Returns (customer, false) if the status is not active or initial.
// Returns (nil, true) if customer-manager is unavailable (fail-open).
func (h *callHandler) ValidateCustomerStatusIncoming(ctx context.Context, customerID uuid.UUID) (*cucustomer.Customer, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerStatusIncoming",
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

	if cu.Status != cucustomer.StatusActive && cu.Status != cucustomer.StatusInitial {
		log.Infof("Customer account is not active or initial. Rejecting incoming call. status: %s", cu.Status)
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

// validateOutgoingCallPermission checks whether the given customer is eligible to make
// an outgoing call. It validates:
// 1. Customer is not nil (caller must fetch the customer before calling this function).
// 2. Customer account status is active.
// 3. For PSTN (TypeTel) destinations, customer identity must be verified.
//    Internal system customer IDs bypass the identity verification check.
func (h *callHandler) validateOutgoingCallPermission(ctx context.Context, cu *cucustomer.Customer, destination commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "validateOutgoingCallPermission",
	})

	if cu == nil {
		return fmt.Errorf("customer not available")
	}
	log = log.WithField("customer_id", cu.ID)

	// check customer account status
	if cu.Status != cucustomer.StatusActive {
		log.Infof("Customer account is not active. Rejecting outgoing call. status: %s", cu.Status)
		return fmt.Errorf("customer account is not active")
	}

	// check identity verification for outgoing PSTN calls only
	if destination.Type == commonaddress.TypeTel {
		// bypass for known internal/system customer IDs
		if cu.ID == cucustomer.IDCallManager ||
			cu.ID == cucustomer.IDAIManager ||
			cu.ID == cucustomer.IDSystem ||
			cu.ID == cucustomer.IDBasicRoute {
			log.Debugf("Internal customer ID, bypassing identity verification. customer_id: %s", cu.ID)
			return nil
		}

		if cu.IdentityVerificationStatus != cucustomer.IdentityVerificationStatusVerified {
			log.Infof("Customer identity not verified. Rejecting outgoing PSTN call. status: %s", cu.IdentityVerificationStatus)
			return fmt.Errorf("customer identity verification required for PSTN calls")
		}
	}

	return nil
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

// ValidateDestination returns true if the outbound call destination is allowed
// by the customer's OutboundConfig destination whitelist.
// Only enforced for TypeTel destinations; other types bypass.
// Internal system customer IDs bypass entirely.
func (h *callHandler) ValidateDestination(ctx context.Context, customerID uuid.UUID, config *outboundconfig.OutboundConfig, destination commonaddress.Address) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateDestination",
		"customer_id": customerID,
	})

	if destination.Type != commonaddress.TypeTel {
		return true
	}
	if cucustomer.IsInternalSystemID(customerID) {
		return true
	}

	country := h.getCountry(ctx, destination.Target)
	if country == "" {
		log.Infof("Could not determine country for destination; denying. destination: %s", destination.Target)
		return false
	}

	if config == nil || len(config.DestinationWhitelist) == 0 {
		log.Infof("No outbound config or empty whitelist; denying. customer_id: %s", customerID)
		return false
	}

	for _, allowed := range config.DestinationWhitelist {
		if allowed == country {
			return true
		}
	}

	log.Infof("Destination country %q not in whitelist. customer_id: %s", country, customerID)
	promCallOutboundWhitelistRejectedTotal.WithLabelValues(country).Inc()
	return false
}

// getCountry returns the ISO 3166 alpha-2 country code (lowercase) for the given
// E.164 phone number, or "" if it cannot be determined.
// The leading "+" is stripped before calling the phonenumber library because
// phonenumber.GetISO3166ByNumber expects digits only (no "+").
// withLandLine is set to true so that landline numbers are also recognised.
func (h *callHandler) getCountry(ctx context.Context, number string) string {
	digits := strings.TrimPrefix(number, "+")
	country := phonenumber.GetISO3166ByNumber(digits, true)

	res := strings.ToLower(country.Alpha2)
	return res
}
