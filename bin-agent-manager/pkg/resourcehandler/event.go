package resourcehandler

import (
	"context"
	"encoding/json"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	whwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/sirupsen/logrus"
)

// EventWebhookPublished handles the webhook-manager's webhook_published event
func (h *resourceHandler) EventWebhookPublished(ctx context.Context, w *whwebhook.Webhook) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "EventWebhookPublished",
		"webhook": w,
	})

	// get type
	tmpData, err := json.Marshal(w.Data)
	if err != nil {
		return nil
	}

	data := whwebhook.Data{}
	if errUnmarshal := json.Unmarshal([]byte(tmpData), &data); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the webhook event. err: %v", errUnmarshal)
		return nil
	}

	switch data.Type {

	////////////////////////////////////
	// groupcall
	////////////////////////////////////
	case string(cmgroupcall.EventTypeGroupcallCreated):
		tmp := cmgroupcall.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookGroupcallCreated(ctx, &tmp)

	case string(cmgroupcall.EventTypeGroupcallProgressing), string(cmgroupcall.EventTypeGroupcallHangup):
		tmp := cmgroupcall.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookGroupcallUpdated(ctx, &tmp)

	case string(cmgroupcall.EventTypeGroupcallDeleted):
		tmp := cmgroupcall.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookGroupcallDeleted(ctx, &tmp)

	////////////////////////////////////
	// call
	////////////////////////////////////
	case string(cmcall.EventTypeCallCreated):
		tmp := cmcall.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookCallCreated(ctx, &tmp)

	case string(cmcall.EventTypeCallCanceling),
		string(cmcall.EventTypeCallDialing),
		string(cmcall.EventTypeCallHangup),
		string(cmcall.EventTypeCallProgressing),
		string(cmcall.EventTypeCallRinging),
		string(cmcall.EventTypeCallTerminating),
		string(cmcall.EventTypeCallUpdated):
		tmp := cmcall.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookCallUpdated(ctx, &tmp)

	case string(cmcall.EventTypeCallDeleted):
		tmp := cmcall.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookCallDeleted(ctx, &tmp)

	////////////////////////////////////
	// chatroom
	////////////////////////////////////
	case string(chatchatroom.EventTypeChatroomCreated):
		tmp := chatchatroom.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookChatroomCreated(ctx, &tmp)

	case string(chatchatroom.EventTypeChatroomUpdated):
		tmp := chatchatroom.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookChatroomUpdated(ctx, &tmp)

	case string(chatchatroom.EventTypeChatroomDeleted):
		tmp := chatchatroom.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookChatroomDeleted(ctx, &tmp)

	////////////////////////////////////
	// messagechatroom
	////////////////////////////////////
	case string(chatmessagechatroom.EventTypeMessagechatroomCreated):
		tmp := chatmessagechatroom.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookMessagechatroomCreated(ctx, &tmp)

	case string(chatmessagechatroom.EventTypeMessagechatroomUpdated):
		tmp := chatmessagechatroom.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookMessagechatroomUpdated(ctx, &tmp)

	case string(chatmessagechatroom.EventTypeMessagechatroomDeleted):
		tmp := chatmessagechatroom.WebhookMessage{}
		if errUnmarshal := json.Unmarshal([]byte(data.Data), &tmp); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the webhook data. err: %v", errUnmarshal)
			return nil
		}
		return h.eventWebhookMessagechatroomDeleted(ctx, &tmp)

	////////////////////////////////////
	// unsupported event
	////////////////////////////////////
	default:
		// log.Errorf("Unknown webhook event type. type: %s", data.Type)
		//
		return nil
	}
}
