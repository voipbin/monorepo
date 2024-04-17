package chatbotcallhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/service"
)

// ServiceStart is creating a new chatbotcall.
// it increases corresponded counter
func (h *chatbotcallHandler) ServiceStart(
	ctx context.Context,
	customerID uuid.UUID,
	chatbotID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType chatbotcall.ReferenceType,
	referenceID uuid.UUID,
	gender chatbotcall.Gender,
	language string,
) (*service.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceStart",
		"customer_id":    customerID,
		"chatbot_id":     chatbotID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"gender":         gender,
		"language":       language,
	})

	// get chatbot
	c, err := h.chatbotHandler.Get(ctx, chatbotID)
	if err != nil {
		log.Errorf("Could not get chatbot. err: %v", err)
		return nil, errors.Wrap(err, "could not get chatbot info")
	}

	// create confbridge
	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, customerID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	// create chatbotcall
	cc, err := h.Create(ctx, customerID, c.ID, c.EngineType, activeflowID, referenceType, referenceID, cb.ID, gender, language)
	if err != nil {
		log.Errorf("Could not create chatbotcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create chatbotcall.")
	}
	log.WithField("chatbotcall", cc).Debugf("Created chatbotcall. chatbotcall_id: %s", cc.ID)

	go func(c *chatbot.Chatbot, cc *chatbotcall.Chatbotcall) {
		if errInit := h.ChatInit(ctx, c, cc); errInit != nil {
			log.Errorf("Could not init chatbotcall. err: %v", errInit)
		}
	}(c, cc)

	// create push actions for service start
	optJoin := fmaction.OptionConfbridgeJoin{
		ConfbridgeID: cb.ID,
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
}
