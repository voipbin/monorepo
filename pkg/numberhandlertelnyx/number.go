package numberhandlertelnyx

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// CreateNumber creates a new order numbers of given numbers from the telnyx
func (h *numberHandlerTelnyx) CreateNumber(customerID uuid.UUID, num string, flowID uuid.UUID, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "CreateNumber",
			"customer_id": customerID,
			"number":      num,
		},
	)

	// send a request to number providers
	numbers := []string{num}
	resOrder, err := h.requestExternal.TelnyxNumberOrdersPost(token, numbers, connectionID, messagingProfileID)
	if err != nil {
		log.Errorf("Could not send the order request to the telnyx. err: %v", err)
		return nil, err
	}

	// get ordered number
	tmpNum, err := h.requestExternal.TelnyxPhoneNumbersIDGet(resOrder.PhoneNumbers[0].ID, token)
	if err != nil {
		log.Errorf("Could not get ordered number. phone_number_id: %s, err: %v", resOrder.PhoneNumbers[0].ID, err)
		return nil, err
	}

	res := tmpNum.ConvertNumber()
	return res, nil
}

// ReleaseNumber release an existed order number from the telnyx
func (h *numberHandlerTelnyx) ReleaseNumber(ctx context.Context, number *number.Number) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "ReleaseNumber",
			"number": number,
		},
	)

	// delete the number from the telnyx
	phoneNumber, err := h.requestExternal.TelnyxPhoneNumbersIDDelete(token, number.ProviderReferenceID)
	if err != nil {
		log.Errorf("Could not delete the number from the telnyx. number: %s, err: %v", number.ID, err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"phone_number": phoneNumber,
		},
	).Debugf("Release the number from the telnyx correctly. number: %s", number.ID)

	return nil
}
