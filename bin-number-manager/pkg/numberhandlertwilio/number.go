package numberhandlertwilio

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

// classifyOrderError classifies the error of the purchase request. Only an HTTP answer that
// definitely refuses the request is a rejection; every other error (network, timeout, 5xx,
// 408, 409, 429, 3xx, ...) may have created the number and is unconfirmed. The answer is
// classified by its HTTP status only; the twilio error code is for the log.
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

// NumberPurchase purchases the number from the twilio. The twilio answers a purchase with the
// sid of the new number at once, so there is no lookup after the order.
func (h *numberHandlerTwilio) NumberPurchase(num string) (*providernumber.ProviderNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "NumberPurchase",
		"number": num,
	})

	// nothing is sent for these, so the next provider may be tried
	if !regexpE164.MatchString(num) {
		log.Errorf("The number is not in the E.164 format.")
		return nil, fmt.Errorf("%w: the number is not in the E.164 format", provider.ErrPurchaseRejected)
	}
	if !regexpTrunkSID.MatchString(h.trunkSID) {
		log.Errorf("PROVIDER_CONFIG_INVALID. The twilio trunk_sid is not set or malformed.")
		return nil, fmt.Errorf("%w: the twilio trunk_sid is not set or malformed", provider.ErrPurchaseRejected)
	}

	res, err := h.requestExternal.TwilioIncomingPhoneNumbersPost(h.sid, h.token, num, h.trunkSID)
	if err != nil {
		var statusErr *requestexternal.HTTPStatusError
		if stderrors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden || statusErr.StatusCode == http.StatusNotFound) {
			// the account setting is wrong: do not let the fallback hide it
			log.Errorf("PROVIDER_AUTH_REJECTED. The twilio refused the credential or the account. status: %d, code: %d", statusErr.StatusCode, statusErr.ErrorCode)
		}
		log.Errorf("Could not purchase the number from the twilio. err: %v", err)
		return nil, classifyOrderError(err)
	}

	// the answer is a success but it does not tell the number: it may exist, so do not fall back
	if !regexpNumberSID.MatchString(res.SID) {
		log.Errorf("PURCHASE_UNCONFIRMED. The twilio answered without a valid number sid.")
		return nil, fmt.Errorf("%w: the twilio answered without a valid number sid", provider.ErrPurchaseUnconfirmed)
	}

	return res.ConvertProviderNumber(), nil
}

// NumberRelease releases the number from the twilio account.
func (h *numberHandlerTwilio) NumberRelease(ctx context.Context, num *number.Number) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "NumberRelease",
		"number_id": num.ID,
	})

	if !regexpNumberSID.MatchString(num.ProviderReferenceID) {
		log.Errorf("The provider reference id is not a twilio number sid.")
		return fmt.Errorf("the provider reference id is not a twilio number sid")
	}

	if err := h.requestExternal.TwilioIncomingPhoneNumbersIDDelete(h.sid, h.token, num.ProviderReferenceID); err != nil {
		log.Errorf("Could not release the number from the twilio. err: %v", err)
		return err
	}
	log.Debugf("Released the number from the twilio.")

	return nil
}

// NumberUpdateTags does nothing: the twilio has no tags (a friendly name is a single short
// string and is not used instead). The caller treats a tag failure as a log only, so a no-op
// avoids a false error.
func (h *numberHandlerTwilio) NumberUpdateTags(ctx context.Context, num *number.Number, tags []string) error {
	logrus.WithFields(logrus.Fields{
		"func":      "NumberUpdateTags",
		"number_id": num.ID,
	}).Debugf("The twilio has no tags. Skipping.")

	return nil
}
