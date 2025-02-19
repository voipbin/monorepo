package chatbotcallhandler

import (
	"context"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *chatbotcallHandler) Start(
	ctx context.Context,
	chatbotID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType chatbotcall.ReferenceType,
	referenceID uuid.UUID,
	gender chatbotcall.Gender,
	language string,
) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"chatbot_id":     chatbotID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"gender":         gender,
		"language":       language,
	})

	c, err := h.chatbotHandler.Get(ctx, chatbotID)
	if err != nil {
		log.Errorf("Could not get chatbot. err: %v", err)
		return nil, errors.Wrap(err, "could not get chatbot info")
	}

	switch referenceType {
	case chatbotcall.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, c, activeflowID, referenceID, gender, language)

	case chatbotcall.ReferenceTypeNone:
		return h.startReferenceTypeNone(ctx, c, gender, language)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return nil, errors.New("unsupported reference type")
	}
}

func (h *chatbotcallHandler) startReferenceTypeCall(ctx context.Context, c *chatbot.Chatbot, activeflowID uuid.UUID, referenceID uuid.UUID, gender chatbotcall.Gender, language string) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeCall",
	})

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, c.CustomerID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	res, err := h.Create(ctx, c, activeflowID, chatbotcall.ReferenceTypeCall, referenceID, cb.ID, gender, language)
	if err != nil {
		log.Errorf("Could not create chatbotcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create chatbotcall.")
	}
	log.WithField("chatbotcall", res).Debugf("Created chatbotcall. chatbotcall_id: %s", res.ID)

	if errInit := h.ChatInit(ctx, c, res); errInit != nil {
		log.Errorf("Could not initialize chat. err: %v", errInit)
		return nil, errors.Wrap(errInit, "Could not initialize chat")
	}

	return res, nil
}

func (h *chatbotcallHandler) startReferenceTypeNone(ctx context.Context, c *chatbot.Chatbot, gender chatbotcall.Gender, language string) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeCall",
	})

	res, err := h.Create(ctx, c, uuid.Nil, chatbotcall.ReferenceTypeCall, uuid.Nil, uuid.Nil, gender, language)
	if err != nil {
		log.Errorf("Could not create chatbotcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create chatbotcall.")
	}
	log.WithField("chatbotcall", res).Debugf("Created chatbotcall. chatbotcall_id: %s", res.ID)

	go h.ChatInit(ctx, c, res)

	return res, nil
}
