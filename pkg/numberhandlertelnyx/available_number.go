package numberhandlertelnyx

import (
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
)

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandlerTelnyx) GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {

	// send a request number providers

	// telnyx
	tmpNumbers, err := h.requestExternal.TelnyxAvailableNumberGets(countyCode, "", "", limit)
	if err != nil {
		logrus.Errorf("Could not get available numbers from the telnyx. err: %v", err)
		return nil, err
	}

	// twilio

	// messagebird

	res := []*availablenumber.AvailableNumber{}
	for _, tmp := range tmpNumbers {
		res = append(res, tmp.ConvertAvailableNumber())
	}

	return res, nil
}
