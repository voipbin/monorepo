package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *aicallHandler) Start(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"ai_id":          aiID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"gender":         gender,
		"language":       language,
	})

	c, err := h.aiHandler.Get(ctx, aiID)
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
		return nil, errors.Wrap(err, "could not get ai info")
	}

	switch referenceType {
	case aicall.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, c, activeflowID, referenceID, gender, language)

	case aicall.ReferenceTypeNone:
		return h.startReferenceTypeNone(ctx, c, gender, language)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return nil, errors.New("unsupported reference type")
	}
}

func (h *aicallHandler) startReferenceTypeCall(ctx context.Context, c *ai.AI, activeflowID uuid.UUID, referenceID uuid.UUID, gender aicall.Gender, language string) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeCall",
	})

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, c.CustomerID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	res, err := h.Create(ctx, c, activeflowID, aicall.ReferenceTypeCall, referenceID, cb.ID, gender, language)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	go func(cctx context.Context) {
		if errInit := h.chatInit(cctx, c, res); errInit != nil {
			log.Errorf("Could not initialize chat. err: %v", errInit)
		}

	}(context.Background())

	return res, nil
}

func (h *aicallHandler) startReferenceTypeNone(ctx context.Context, c *ai.AI, gender aicall.Gender, language string) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeNone",
	})

	tmp, err := h.Create(ctx, c, uuid.Nil, aicall.ReferenceTypeNone, uuid.Nil, uuid.Nil, gender, language)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", tmp).Debugf("Created aicall. aicall_id: %s", tmp.ID)

	if errInit := h.chatInit(ctx, c, tmp); errInit != nil {
		log.Errorf("Could not initialize chat. err: %v", errInit)
		return nil, errors.Wrap(errInit, "Could not initialize chat")
	}

	res, err := h.UpdateStatusStart(ctx, tmp.ID, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, errors.Wrapf(err, "Could not update the status to start. aicall_id: %s", tmp.ID)
	}

	return res, nil
}
