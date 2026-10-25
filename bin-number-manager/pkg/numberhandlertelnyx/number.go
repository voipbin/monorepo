package numberhandlertelnyx

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/requestexternal"
)

// orderStatusFailure is the status of a number order that the telnyx has definitely failed.
const orderStatusFailure = "failure"

// classifyOrderError classifies the error of the order request. Only an HTTP answer that
// definitely refuses the request is a rejection; every other error (network, timeout, 5xx,
// 408, 409, 429, ...) may have been accepted by the provider and is unconfirmed.
func classifyOrderError(err error) error {
	var statusErr *requestexternal.HTTPStatusError
	if stderrors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusUnprocessableEntity:
			return fmt.Errorf("%w: %w", provider.ErrPurchaseRejected, err)
		}
	}

	return fmt.Errorf("%w: %w", provider.ErrPurchaseUnconfirmed, err)
}

// NumberPurchase creates a new order numbers of given numbers from the telnyx
func (h *numberHandlerTelnyx) NumberPurchase(num string) (*providernumber.ProviderNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "PurchaseNumber",
		"number": num,
	})

	// send a request to number providers
	numbers := []string{num}
	resOrder, err := h.requestExternal.TelnyxNumberOrdersPost(h.token, numbers, h.connectionID, h.profileID)
	if err != nil {
		log.Errorf("Could not send the order request to the telnyx. err: %v", err)
		return nil, classifyOrderError(err)
	}
	log.WithField("ordered_number", resOrder).Debugf("Ordered number correctly. number: %v", resOrder.PhoneNumbers)

	// the order may have been accepted: only an order that the telnyx reports as failed is a definite refusal
	if resOrder.Status == orderStatusFailure {
		log.Errorf("The telnyx reported the order as failed.")
		return nil, fmt.Errorf("%w: the order status is %s", provider.ErrPurchaseRejected, resOrder.Status)
	}

	tmp, err := h.requestExternal.TelnyxPhoneNumbersGetByNumber(h.token, num)
	if err != nil {
		// the order was sent but the result could not be confirmed: do not let the caller try another provider
		log.Errorf("PURCHASE_UNCONFIRMED. Could not get ordered number info. err: %v", err)
		return nil, fmt.Errorf("%w: could not get the ordered number info: %w", provider.ErrPurchaseUnconfirmed, err)
	}

	res := tmp.ConvertProviderNumber()
	return res, nil
}

// NumberRelease release an existed order number from the telnyx
func (h *numberHandlerTelnyx) NumberRelease(ctx context.Context, number *number.Number) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ReleaseNumber",
		"number": number,
	})

	// delete the number from the telnyx
	phoneNumber, err := h.requestExternal.TelnyxPhoneNumbersIDDelete(h.token, number.ProviderReferenceID)
	if err != nil {
		log.Errorf("Could not delete the number from the telnyx. number: %s, err: %v", number.ID, err)
		return err
	}
	log.WithField("phone_number", phoneNumber).Debugf("Release the number from the telnyx correctly. number: %s", number.ID)

	return nil
}

// NumberUpdateTags updates the given number's tags
func (h *numberHandlerTelnyx) NumberUpdateTags(ctx context.Context, number *number.Number, tags []string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "NumberUpdateTags",
		"number": number,
		"tags":   tags,
	})

	data := map[string]interface{}{
		"tags": tags,
	}

	// delete the number from the telnyx
	tmp, err := h.requestExternal.TelnyxPhoneNumbersIDUpdate(h.token, number.ProviderReferenceID, data)
	if err != nil {
		log.Errorf("Could not update the number tag from the telnyx. number_id: %s, err: %v", number.ID, err)
		return err
	}
	log.WithField("phone_number", tmp).Debugf("Updated   Release the number from the telnyx correctly. number: %s", number.ID)

	return nil
}
