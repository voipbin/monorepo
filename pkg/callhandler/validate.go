package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
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

	validBalance, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID)
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
