package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaigncall"
)

// processEventFMActiveflowDeleted handles the flow-manager's activeflow_deleted event.
func (h *subscribeHandler) processEventFMActiveflowDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventFMActiveflowDeleted",
		"event": m,
	})

	c := fmactiveflow.Activeflow{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get campaigncall
	cc, err := h.campaigncallHandler.GetByActiveflowID(ctx, c.ID)
	if err != nil {
		// campaigncall does not exist.
		return nil
	}

	if cc.Status == campaigncall.StatusDone {
		// already done
		return nil
	}

	if cc.ReferenceType == campaigncall.ReferenceTypeCall {
		// will be handled by the processEventCMCallHungup
		return nil
	}

	// campaigncall handle
	_, err = h.campaigncallHandler.EventHandleActiveflowDeleted(ctx, cc)
	if err != nil {
		log.Errorf("Could not handle the event correctly. err: %v", err)
	}

	// campaign handle
	if errEvent := h.campaignHandler.EventHandleActiveflowDeleted(ctx, cc.CampaignID); errEvent != nil {
		log.Errorf("Could not handle the cmcallhangup event correctly by campaign handler. err: %v", errEvent)
	}

	return nil
}
