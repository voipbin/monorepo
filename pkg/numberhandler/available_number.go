package numberhandler

import (
	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
)

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandler) GetAvailableNumbers(countyCode string, limit uint) ([]*models.AvailableNumber, error) {

	// use telnyx as a default
	return h.numHandlerTelnyx.GetAvailableNumbers(countyCode, limit)
}
