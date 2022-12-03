package numberhandler

import "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandler) GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {

	// use telnyx as a default
	return h.numberHandlerTelnyx.GetAvailableNumbers(countyCode, limit)
}
