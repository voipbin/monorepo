package chatbotcallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/chatbot"
	"monorepo/bin-ai-manager/models/chatbotcall"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

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

	go func(cctx context.Context) {
		if errInit := h.chatInit(cctx, c, res); errInit != nil {
			log.Errorf("Could not initialize chat. err: %v", errInit)
		}

	}(context.Background())

	return res, nil
}

func (h *chatbotcallHandler) startReferenceTypeNone(ctx context.Context, c *chatbot.Chatbot, gender chatbotcall.Gender, language string) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeNone",
	})

	tmp, err := h.Create(ctx, c, uuid.Nil, chatbotcall.ReferenceTypeNone, uuid.Nil, uuid.Nil, gender, language)
	if err != nil {
		log.Errorf("Could not create chatbotcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create chatbotcall.")
	}
	log.WithField("chatbotcall", tmp).Debugf("Created chatbotcall. chatbotcall_id: %s", tmp.ID)

	if errInit := h.chatInit(ctx, c, tmp); errInit != nil {
		log.Errorf("Could not initialize chat. err: %v", errInit)
		return nil, errors.Wrap(errInit, "Could not initialize chat")
	}

	res, err := h.UpdateStatusStart(ctx, tmp.ID, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, errors.Wrapf(err, "Could not update the status to start. chatbotcall_id: %s", tmp.ID)
	}

	return res, nil
}
