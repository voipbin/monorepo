package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// chatroommessageGet validates the chatroommessage's ownership and returns the chatroommessage info.
func (h *serviceHandler) chatroommessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":               "chatroommessageGet",
			"customer_id":        u.ID,
			"chatroommessage_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.ChatV1MessagechatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the chatroommessage info. err: %v", err)
		return nil, err
	}
	log.WithField("messagechatroom", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this customer.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatroommessageGet gets the chatroommessage of the given id.
// It returns chatroommessage if it succeed.
func (h *serviceHandler) ChatroommessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroommessageGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Getting a chatroommessage.")

	// get chat
	res, err := h.chatroommessageGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	return res, nil
}

// ChatroommessageGetsByChatroomID gets the list of chatroommessages of the given owner id.
// It returns list of chatroommessages if it succeed.
func (h *serviceHandler) ChatroommessageGetsByChatroomID(ctx context.Context, u *cscustomer.Customer, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroommessageGetsByChatroomID",
		"customer_id": u.ID,
		"username":    u.Username,
		"chatroom_id": chatroomID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chatroommessages.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get owner
	tmp, err := h.chatroomGet(ctx, u, chatroomID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", tmp).Debugf("Found chatroom info. chatroom_id: %s", chatroomID)

	// get chats
	tmps, err := h.reqHandler.ChatV1MessagechatroomGetsByChatroomID(ctx, chatroomID, token, size)
	if err != nil {
		log.Errorf("Could not get chatroommessages info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessages info. err: %v", err)
	}

	// create result
	res := []*chatmessagechatroom.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatroommessageDelete deletes the chatroom.
func (h *serviceHandler) ChatroommessageDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "ChatroommessageDelete",
		"customer_id":        u.ID,
		"username":           u.Username,
		"chatroommessage_id": id,
	})
	log.Debug("Deleting a chatroommessage.")

	// get chatroommessage
	_, err := h.chatroommessageGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1MessagechatroomDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroommessage. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
