package chatbotcallhandler

import (
	"context"
	"encoding/json"
	"fmt"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/service"
)

// ServiceStart is creating a new chatbotcall.
// it increases corresponded counter
func (h *chatbotcallHandler) ServiceStart(
	ctx context.Context,
	chatbotID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType chatbotcall.ReferenceType,
	referenceID uuid.UUID,
	gender chatbotcall.Gender,
	language string,
) (*service.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceStart",
		"chatbot_id":     chatbotID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"gender":         gender,
		"language":       language,
	})

	cc, err := h.Start(ctx, chatbotID, activeflowID, referenceType, referenceID, gender, language)
	if err != nil {
		log.Errorf("Could not start chatbotcall. err: %v", err)
		return nil, fmt.Errorf("could not start chatbotcall. err: %v", err)
	}

	// create push actions for service start
	optJoin := fmaction.OptionConfbridgeJoin{
		ConfbridgeID: cc.ConfbridgeID,
	}
	optString, err := json.Marshal(optJoin)
	if err != nil {
		log.Errorf("Could not marshal the conference join option. err: %v", err)
		return nil, errors.Wrap(err, "Could not marshal the conference join option.")
	}
	actions := []fmaction.Action{
		{
			ID:     h.utilHandler.UUIDCreate(),
			Type:   fmaction.TypeConfbridgeJoin,
			Option: optString,
		},
	}

	res := &service.Service{
		ID:          cc.ID,
		Type:        service.TypeChatbotcall,
		PushActions: actions,
	}

	return res, nil

	// // get chatbot
	// c, err := h.chatbotHandler.Get(ctx, chatbotID)
	// if err != nil {
	// 	log.Errorf("Could not get chatbot. err: %v", err)
	// 	return nil, errors.Wrap(err, "could not get chatbot info")
	// }

	// if c.CustomerID != customerID {
	// 	log.Errorf("Chatbot does not belong to the customer. chatbot_id: %s, customer_id: %s", chatbotID, customerID)
	// 	return nil, fmt.Errorf("chatbot does not belong to the customer. chatbot_id: %s, customer_id: %s", chatbotID, customerID)
	// }

	// switch referenceType {
	// case chatbotcall.ReferenceTypeCall:
	// 	res, err := h.serviceStartReferenceTypeCall(ctx, c, activeflowID, referenceID, gender, language)
	// 	if err != nil {
	// 		log.Errorf("Could not create chatbotcall. err: %v", err)
	// 		return nil, err
	// 	}

	// 	return res, nil

	// default:
	// 	log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
	// 	return nil, fmt.Errorf("unsupported reference type. reference_type: %s", referenceType)
	// }
}

// // ServiceStart is creating a new chatbotcall.
// // it increases corresponded counter
// func (h *chatbotcallHandler) serviceStartReferenceTypeCall(ctx context.Context, c *chatbot.Chatbot, activeflowID uuid.UUID, referenceID uuid.UUID, gender chatbotcall.Gender, language string) (*service.Service, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "ServiceStart",
// 	})

// 	// create confbridge
// 	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, c.CustomerID, cmconfbridge.TypeConference)
// 	if err != nil {
// 		log.Errorf("Could not create confbridge. err: %v", err)
// 		return nil, errors.Wrap(err, "Could not create confbridge")
// 	}

// 	// create chatbotcall
// 	cc, err := h.Create(ctx, c, activeflowID, chatbotcall.ReferenceTypeCall, referenceID, cb.ID, gender, language)
// 	if err != nil {
// 		log.Errorf("Could not create chatbotcall. err: %v", err)
// 		return nil, errors.Wrap(err, "Could not create chatbotcall.")
// 	}
// 	log.WithField("chatbotcall", cc).Debugf("Created chatbotcall. chatbotcall_id: %s", cc.ID)

// 	go h.ChatInit(context.Background(), c, cc)

// 	// create push actions for service start
// 	optJoin := fmaction.OptionConfbridgeJoin{
// 		ConfbridgeID: cb.ID,
// 	}
// 	optString, err := json.Marshal(optJoin)
// 	if err != nil {
// 		log.Errorf("Could not marshal the conference join option. err: %v", err)
// 		return nil, errors.Wrap(err, "Could not marshal the conference join option.")
// 	}
// 	actions := []fmaction.Action{
// 		{
// 			ID:     h.utilHandler.UUIDCreate(),
// 			Type:   fmaction.TypeConfbridgeJoin,
// 			Option: optString,
// 		},
// 	}

// 	res := &service.Service{
// 		ID:          cc.ID,
// 		Type:        service.TypeChatbotcall,
// 		PushActions: actions,
// 	}

// 	return res, nil

// }
