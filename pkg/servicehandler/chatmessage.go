package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// chatmessageGet validates the chatmessage's ownership and returns the chat info.
func (h *serviceHandler) chatmessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "chatmessageGet",
			"customer_id": u.ID,
			"chat_id":     id,
		},
	)

	// send request
	tmp, err := h.reqHandler.ChatV1MessagechatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the messagechat info. err: %v", err)
		return nil, err
	}
	log.WithField("messagechat", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatmessageCreate is a service handler for chatmessage creation.
func (h *serviceHandler) ChatmessageCreate(
	ctx context.Context,
	u *cscustomer.Customer,
	chatID uuid.UUID,
	source commonaddress.Address,
	messageType chatmessagechat.Type,
	text string,
	medias []chatmedia.Media,
) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatmessageCreate",
		"customer_id": u.ID,
	})

	log.Debug("Creating a new messagechat.")
	tmp, err := h.reqHandler.ChatV1MessagechatCreate(
		ctx,
		u.ID,
		chatID,
		source,
		messageType,
		text,
		medias,
	)
	if err != nil {
		log.Errorf("Could not create a new messagechat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatmessageGetsByChatID gets the list of chats of the given chat id.
// It returns list of messagechats if it succeed.
func (h *serviceHandler) ChatmessageGetsByChatID(ctx context.Context, u *cscustomer.Customer, chatID uuid.UUID, size uint64, token string) ([]*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatmessageGetsByChatID",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     chatID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a messagechats.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if size == 0 {
		size = 10
	}

	// get chat info
	tmp, err := h.chatGet(ctx, u, chatID)
	if err != nil {
		log.Errorf("Could not get chat info. err: %v", err)
		return nil, err
	}
	log.WithField("chat", tmp).Debugf("Found chat info. chat_id: %s", tmp.ID)

	// get chats
	tmps, err := h.reqHandler.ChatV1MessagechatGetsByChatID(ctx, chatID, token, size)
	if err != nil {
		log.Errorf("Could not get chats info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*chatmessagechat.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatmessageGet gets the chatmessage of the given id.
// It returns chat if it succeed.
func (h *serviceHandler) ChatmessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ChatmessageGet",
		"customer_id":    u.ID,
		"username":       u.Username,
		"chatmessage_id": id,
	})
	log.Debug("Getting a ChatmessageGet.")

	// get chatmessage
	res, err := h.chatmessageGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatmessage info. err: %v", err)
	}

	return res, nil
}

// ChatmessageDelete deletes the chatmessage.
func (h *serviceHandler) ChatmessageDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ChatmessageDelete",
		"customer_id":    u.ID,
		"username":       u.Username,
		"chatmessage_id": id,
	})
	log.Debug("Deleting a chatmessage.")

	// get chatmessage
	_, err := h.chatmessageGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chatmessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatmessage info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1MessagechatDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatmessage. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
