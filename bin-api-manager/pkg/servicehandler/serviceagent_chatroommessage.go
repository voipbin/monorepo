package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentChatroommessageGet gets the chatroommessage of the given id.
// It returns chatroommessage if it succeed.
func (h *serviceHandler) ServiceAgentChatroommessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroommessageGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Getting a chatroommessage.")

	// get chat
	tmp, err := h.chatroommessageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentChatroommessageGets sends a request to chat-manager
// to getting the given chatroom's list of chatroom message.
// it returns list of chatroom messages if it succeed.
func (h *serviceHandler) ServiceAgentChatroommessageGets(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroommessageGets",
		"agent":       a,
		"chatroom_id": chatroomID,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmp, err := h.chatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", tmp).Debugf("Found chatroom info. chatroom_id: %s", chatroomID)

	if tmp.OwnerID != a.ID {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// filters
	filters := map[string]string{
		"chatroom_id": chatroomID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	res, err := h.chatroommessageGetsWithFilters(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not chatrooms info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// // ServiceAgentChatroomGets sends a request to chat-manager
// // to getting the given agent's list of chatrooms.
// // it returns list of chatrooms if it succeed.
// func (h *serviceHandler) ServiceAgentChatroomGet(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID) (*chatroom.WebhookMessage, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":        "ServiceAgentChatroomGets",
// 		"agent":       a,
// 		"chatroom_id": chatroomID,
// 	})

// 	tmp, err := h.chatroomGet(ctx, chatroomID)
// 	if err != nil {
// 		log.Errorf("Could not get chatroom info. err: %v", err)
// 		return nil, err
// 	}

// 	if a.ID != tmp.OwnerID {
// 		return nil, fmt.Errorf("user has no permission")
// 	}

// 	res := tmp.ConvertWebhookMessage()
// 	return res, nil
// }

// // ServiceAgentChatroomDelete sends a request to chat-manager
// // to getting the given agent's list of chatrooms.
// // it returns list of chatrooms if it succeed.
// func (h *serviceHandler) ServiceAgentChatroomDelete(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID) (*chatroom.WebhookMessage, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":        "ServiceAgentChatroomDelete",
// 		"agent":       a,
// 		"chatroom_id": chatroomID,
// 	})

// 	tmp, err := h.chatroomGet(ctx, chatroomID)
// 	if err != nil {
// 		log.Errorf("Could not get chatroom info. err: %v", err)
// 		return nil, err
// 	}

// 	if a.ID != tmp.OwnerID {
// 		return nil, fmt.Errorf("user has no permission")
// 	}

// 	tmpRes, err := h.chatroomDelete(ctx, chatroomID)
// 	if err != nil {
// 		log.Errorf("Could not delete chatroom info. err: %v", err)
// 		return nil, err
// 	}

// 	res := tmpRes.ConvertWebhookMessage()
// 	return res, nil
// }

// // ServiceAgentChatroomCreate creates the chatroom message of the given chatroom id.
// // It returns created chatroommessages if it succeed.
// func (h *serviceHandler) ServiceAgentChatroomCreate(ctx context.Context, a *amagent.Agent, participantIDs []uuid.UUID, name string, detail string) (*chatchatroom.WebhookMessage, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":            "ServiceAgentChatroomCreate",
// 		"agent":           a,
// 		"participant_ids": participantIDs,
// 		"name":            name,
// 		"detail":          detail,
// 	})

// 	res, err := h.ChatroomCreate(ctx, a, participantIDs, name, detail)
// 	if err != nil {
// 		log.Errorf("Could not create chatroom info. err: %v", err)
// 		return nil, err
// 	}

// 	return res, nil
// }
