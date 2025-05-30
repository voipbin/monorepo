package numberhandler

import "monorepo/bin-number-manager/models/availablenumber"

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandler) GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {

	// use telnyx as a default
	return h.numberHandlerTelnyx.GetAvailableNumbers(countyCode, limit)
}
