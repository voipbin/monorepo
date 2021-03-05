package numberhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// RemoveNumbersFlowID removes the flow_id from the all of Numbers
func (h *numberHandler) RemoveNumbersFlowID(ctx context.Context, flowID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"flow_id": flowID,
		},
	)
	log.Debugf("RemoveNumbersFlowID. flow_id: %s", flowID)

	for {
		numbs, err := h.db.NumberGetsByFlowID(ctx, flowID, 100, getCurTime())
		if err != nil || len(numbs) <= 0 {
			break
		}

		for _, numb := range numbs {
			numb.FlowID = uuid.Nil
			if err := h.db.NumberUpdate(ctx, numb); err != nil {
				log.Errorf("Could not remove flow_id. err: %v", err)
			}
		}

		if len(numbs) < 100 {
			// no more list
			break
		}
	}

	return nil
}
