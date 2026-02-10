package numberhandler

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
)

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandler) GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {

	// use telnyx as a default
	return h.numberHandlerTelnyx.GetAvailableNumbers(countyCode, limit)
}

// GetAvailableVirtualNumbers generates random available virtual numbers that are not already registered.
func (h *numberHandler) GetAvailableVirtualNumbers(ctx context.Context, limit uint) ([]*availablenumber.AvailableNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "GetAvailableVirtualNumbers",
		"limit": limit,
	})
	log.Debugf("Getting available virtual numbers. limit: %d", limit)

	if limit == 0 {
		limit = 10
	}

	// generate 3x candidates to account for collisions
	candidateCount := int(limit) * 3
	candidates := make([]string, 0, candidateCount)
	seen := make(map[string]bool)

	for len(candidates) < candidateCount {
		// generate random number in range +999001000000 to +999999999999
		// area codes 001-999 (excluding 000 which is reserved)
		areaCode := rand.Intn(999) + 1        // 1-999
		subscriber := rand.Intn(1000000)       // 000000-999999
		num := fmt.Sprintf("+999%03d%06d", areaCode, subscriber)

		if seen[num] {
			continue
		}
		seen[num] = true
		candidates = append(candidates, num)
	}

	// check which candidates are already taken
	existing, err := h.db.NumberGetExistingNumbers(ctx, candidates)
	if err != nil {
		log.Errorf("Could not check existing numbers. err: %v", err)
		return nil, fmt.Errorf("could not check existing numbers: %w", err)
	}

	existingSet := make(map[string]bool, len(existing))
	for _, n := range existing {
		existingSet[n] = true
	}

	// filter out taken numbers and build result
	res := make([]*availablenumber.AvailableNumber, 0, limit)
	for _, num := range candidates {
		if existingSet[num] {
			continue
		}

		res = append(res, &availablenumber.AvailableNumber{
			Number:       num,
			ProviderName: number.ProviderNameNone,
			Country:      "999",
			Features:     []availablenumber.Feature{availablenumber.FeatureVoice},
		})

		if uint(len(res)) >= limit {
			break
		}
	}

	log.Debugf("Found %d available virtual numbers.", len(res))
	return res, nil
}
