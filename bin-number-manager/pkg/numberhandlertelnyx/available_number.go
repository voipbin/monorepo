package numberhandlertelnyx

import (
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/availablenumber"
)

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandlerTelnyx) GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GetAvailableNumbers",
		"country_code": countyCode,
		"limit":        limit,
	})

	// send a request number providers
	tmpNumbers, err := h.requestExternal.TelnyxAvailableNumberGets(defaultToken, countyCode, "", "", limit)
	if err != nil {
		log.Errorf("Could not get available numbers from the telnyx. err: %v", err)
		return nil, err
	}

	res := []*availablenumber.AvailableNumber{}
	for _, tmp := range tmpNumbers {
		res = append(res, tmp.ConvertAvailableNumber())
	}

	return res, nil
}
