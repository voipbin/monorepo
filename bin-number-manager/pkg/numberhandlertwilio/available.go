package numberhandlertwilio

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/availablenumber"
)

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandlerTwilio) GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GetAvailableNumbers",
		"country_code": countyCode,
		"limit":        limit,
	})
	log.Errorf("Unimplemented: GetAvailableNumbers")

	return nil, fmt.Errorf("unimplemented: GetAvailableNumbers")
}
