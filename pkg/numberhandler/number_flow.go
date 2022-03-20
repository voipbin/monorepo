package numberhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
)

// RemoveNumbersFlowID removes the flow_id from the all of Numbers
func (h *numberHandler) RemoveNumbersFlowID(ctx context.Context, flowID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"flow_id": flowID,
		},
	)
	log.Debugf("RemoveNumbersFlowID. flow_id: %s", flowID)

	// removing call_flow_id
	for {
		numbs, err := h.db.NumberGetsByCallFlowID(ctx, flowID, 100, dbhandler.GetCurTime())
		if err != nil || len(numbs) <= 0 {
			break
		}

		for _, numb := range numbs {
			log.WithField("number", numb).Debugf("Removing call_flow_id from the number. number_id: %s, number_number: %s", numb.ID, numb.Number)
			if err := h.db.NumberUpdateCallFlowID(ctx, numb.ID, uuid.Nil); err != nil {
				log.Errorf("Could not remove flow_id. err: %v", err)
			}
		}

		if len(numbs) < 100 {
			// no more list
			break
		}
	}

	// removing message_flow_id
	for {
		numbs, err := h.db.NumberGetsByMessageFlowID(ctx, flowID, 100, dbhandler.GetCurTime())
		if err != nil || len(numbs) <= 0 {
			break
		}

		for _, numb := range numbs {
			log.WithField("number", numb).Debugf("Removing message_flow_id from the number. number_id: %s, number_number: %s", numb.ID, numb.Number)
			if err := h.db.NumberUpdateMessageFlowID(ctx, numb.ID, uuid.Nil); err != nil {
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
