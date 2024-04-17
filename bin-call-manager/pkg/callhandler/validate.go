package callhandler

import (
	"context"
	"strings"

	"monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/dongri/phonenumber"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
)

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
	validBalance, err := h.reqHandler.CustomerV1CustomerIsValidBalance(ctx, customerID, billing.ReferenceTypeCall, country, 1)

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
