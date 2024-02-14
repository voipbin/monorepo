package numberhandlertelnyx

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/providernumber"
)

// PurchaseNumber creates a new order numbers of given numbers from the telnyx
func (h *numberHandlerTelnyx) PurchaseNumber(num string) (*providernumber.ProviderNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "PurchaseNumber",
		"number": num,
	})

	// send a request to number providers
	numbers := []string{num}
	resOrder, err := h.requestExternal.TelnyxNumberOrdersPost(defaultToken, numbers, defaultConnectionID, defaultMessagingProfileID)
	if err != nil {
		log.Errorf("Could not send the order request to the telnyx. err: %v", err)
		return nil, errors.Wrap(err, "could not send the order request to the telnyx")
	}
	log.WithField("ordered_number", resOrder).Debugf("Ordered number correctly. number: %v", resOrder.PhoneNumbers)

	tmp, err := h.requestExternal.TelnyxPhoneNumbersGetByNumber(defaultToken, num)
	if err != nil {
		log.Errorf("Could not get ordered number info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertProviderNumber()
	return res, nil
}

// ReleaseNumber release an existed order number from the telnyx
func (h *numberHandlerTelnyx) ReleaseNumber(ctx context.Context, number *number.Number) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ReleaseNumber",
		"number": number,
	})

	// delete the number from the telnyx
	phoneNumber, err := h.requestExternal.TelnyxPhoneNumbersIDDelete(defaultToken, number.ProviderReferenceID)
	if err != nil {
		log.Errorf("Could not delete the number from the telnyx. number: %s, err: %v", number.ID, err)
		return err
	}
	log.WithField("phone_number", phoneNumber).Debugf("Release the number from the telnyx correctly. number: %s", number.ID)

	return nil
}
