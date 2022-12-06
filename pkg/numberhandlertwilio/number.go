package numberhandlertwilio

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// CreateNumber creates a new order numbers of given numbers from the telnyx
func (h *numberHandlerTwilio) CreateNumber(customerID uuid.UUID, num string, flowID uuid.UUID, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "CreateNumber",
			"customer_id": customerID,
			"number":      num,
		},
	)

	log.Errorf("Unimplemented: CreateNumber")
	return nil, fmt.Errorf("unimplemented: CreateNumber")
}

// ReleaseNumber release an existed order number from the telnyx
func (h *numberHandlerTwilio) ReleaseNumber(ctx context.Context, number *number.Number) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "ReleaseNumber",
			"number": number,
		},
	)

	log.Errorf("Unimplemented: ReleaseNumber")
	return fmt.Errorf("unimplemented: ReleaseNumber")
}
