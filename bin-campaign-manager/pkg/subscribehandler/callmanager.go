package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processEventCMCallHungup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHungup(ctx context.Context, m *sock.Event) error {
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
	// the campaign handle below must run even if this fails (the campaigncall may already be done and the campaign
	// may still need its stop check), so it uses the campaign id of the campaigncall loaded above.
	if _, err = h.campaigncallHandler.EventHandleReferenceCallHungup(ctx, &c, cc); err != nil {
		log.Errorf("Could not handle the event correctly. err: %v", err)
	}

	// campaign handle
	if errEvent := h.campaignHandler.EventHandleReferenceCallHungup(ctx, cc.CampaignID); errEvent != nil {
		log.Errorf("Could not handle the cmcallhangup event correctly by campaign handler. err: %v", errEvent)
	}

	return nil
}
