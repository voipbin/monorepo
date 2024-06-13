package agenthandler

import (
	"context"
	"encoding/json"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	whwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/sirupsen/logrus"
)

type Data struct {
	Type string `json:"type"`
}

// EventWebhookPublished handles the webhook-manager's webhook_published event
func (h *agentHandler) EventWebhookPublished(ctx context.Context, w *whwebhook.Webhook) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "EventWebhookPublished",
		"webhook": w,
	})

	// get type
	data := Data{}
	if errUnmarshal := json.Unmarshal([]byte(w.Data.(string)), &data); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the webhook event. err: %v", errUnmarshal)
		return nil
	}

	switch data.Type {

	////////////////////////////////////
	// groupcall
	////////////////////////////////////
	case string(cmgroupcall.EventTypeGroupcallCreated):
		tmp := cmgroupcall.Groupcall{}
		if errUnmarshal := json.Unmarshal([]byte(w.Data.(string)), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.webhookGroupcallCreated(ctx, &tmp)

	case string(cmgroupcall.EventTypeGroupcallProgressing), string(cmgroupcall.EventTypeGroupcallHangup):
		tmp := cmgroupcall.Groupcall{}
		if errUnmarshal := json.Unmarshal([]byte(w.Data.(string)), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.webhookGroupcallUpdated(ctx, &tmp)

	////////////////////////////////////
	// call
	////////////////////////////////////
	case string(cmcall.EventTypeCallCreated):
		tmp := cmcall.Call{}
		if errUnmarshal := json.Unmarshal([]byte(w.Data.(string)), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.webhookCallCreated(ctx, &tmp)

	case string(cmcall.EventTypeCallCanceling),
		string(cmcall.EventTypeCallDialing),
		string(cmcall.EventTypeCallHangup),
		string(cmcall.EventTypeCallProgressing),
		string(cmcall.EventTypeCallRinging),
		string(cmcall.EventTypeCallTerminating),
		string(cmcall.EventTypeCallUpdated):
		tmp := cmcall.Call{}
		if errUnmarshal := json.Unmarshal([]byte(w.Data.(string)), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.webhookCallUpdated(ctx, &tmp)

	default:
		log.Errorf("Unknown webhook event type. type: %s", data.Type)
		return nil
	}

}
