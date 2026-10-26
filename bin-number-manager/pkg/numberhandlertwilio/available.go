package numberhandlertwilio

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/availablenumber"
)

// GetAvailableNumbers gets the local numbers of the country from the twilio
func (h *numberHandlerTwilio) GetAvailableNumbers(countryCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GetAvailableNumbers",
		"country_code": countryCode,
		"limit":        limit,
	})

	if !regexpCountry.MatchString(countryCode) {
		return nil, fmt.Errorf("the country code is malformed")
	}

	tmpNumbers, err := h.requestExternal.TwilioAvailableNumberGets(h.sid, h.token, countryCode, limit)
	if err != nil {
		log.Errorf("Could not get available numbers from the twilio. err: %v", err)
		return nil, err
	}

	res := []*availablenumber.AvailableNumber{}
	for _, tmp := range tmpNumbers {
		res = append(res, tmp.ConvertAvailableNumber())
	}

	return res, nil
}
