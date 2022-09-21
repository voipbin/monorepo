package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// chatroomGet validates the chatroom's ownership and returns the chatroom info.
func (h *serviceHandler) chatroomGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "chatroomGet",
			"customer_id": u.ID,
			"chatroom_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.ChatV1ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the chatroom info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this customer.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatroomGetsByOwnerID gets the list of chatrooms of the given owner id.
// It returns list of chatrooms if it succeed.
func (h *serviceHandler) ChatroomGetsByOwnerID(ctx context.Context, u *cscustomer.Customer, ownerID uuid.UUID, size uint64, token string) ([]*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomGetsByOwnerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"owner_id":    ownerID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chatrooms.")

	if token == "" {
		token = getCurTime()
	}

	// get owner
	owner, err := h.agentGet(ctx, u, ownerID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("owner", owner).Debugf("Found owner info. owner_id: %s", owner.ID)

	// get chats
	tmps, err := h.reqHandler.ChatV1ChatroomGetsByOwnerID(ctx, ownerID, token, size)
	if err != nil {
		log.Errorf("Could not get chats info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*chatchatroom.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatroomGet gets the chatroom of the given id.
// It returns chatroom if it succeed.
func (h *serviceHandler) ChatroomGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"chatroom_id": id,
	})
	log.Debug("Getting a chatroom.")

	// get chat
	res, err := h.chatroomGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chatroom info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroom info. err: %v", err)
	}

	return res, nil
}

// ChatroomDelete deletes the chatroom.
func (h *serviceHandler) ChatroomDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"chatroom_id": id,
	})
	log.Debug("Deleting a chat.")

	// get chat
	_, err := h.chatroomGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1ChatroomDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
