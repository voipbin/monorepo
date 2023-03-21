package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallHungup handles the call-manager's confbridge_leaved event.
func (h *subscribeHandler) processEventCMCallHungup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallHungup",
		"event": m,
	})

	c := cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get campaigncall
	cc, err := h.campaigncallHandler.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// campaigncall does not exist.
		return nil
	}

	// campaigncall handle
	newCC, err := h.campaigncallHandler.EventHandleReferenceCallHungup(ctx, &c, cc)
	if err != nil {
		log.Errorf("Could not handle the event correctly. err: %v", err)
	}

	// campaign handle
	if errEvent := h.campaignHandler.EventHandleReferenceCallHungup(ctx, newCC.CampaignID); errEvent != nil {
		log.Errorf("Could not handle the cmcallhangup event correctly by campaign handler. err: %v", errEvent)
	}

	return nil
}
