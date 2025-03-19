package aicallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

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
	resume bool,
) (*aicall.AIcall, error) {

	if resume {
		return h.StartResume(ctx, activeflowID)
	} else {
		return h.StartNew(ctx, aiID, activeflowID, referenceType, referenceID, gender, language)
	}

}

func (h *aicallHandler) StartResume(ctx context.Context, activeflowID uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startContinued",
	})

	// get existing aicall info
	vars, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the activeflow variables. activeflow_id: %s", activeflowID)
	}

	aicallID := uuid.FromStringOrNil(vars.Variables[variableAIcallID])
	if aicallID == uuid.Nil {
		return nil, errors.New("could not get the aicall id from the activeflow variables")
	}

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	// update aicall's confbridge info
	res, err := h.UpdateStatusResuming(ctx, aicallID, cb.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status to resuming. aicall_id: %s", aicallID)
	}

	return res, nil
}

func (h *aicallHandler) StartNew(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startNew",
		"ai_id":         aiID,
		"activeflow_id": activeflowID,
	})
	log.Debugf("Starting a new aicall")

	c, err := h.aiHandler.Get(ctx, aiID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ai info")
	}

	switch referenceType {
	case aicall.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, c, activeflowID, referenceID, gender, language)

	case aicall.ReferenceTypeNone:
		return h.startReferenceTypeNone(ctx, c, gender, language)

	default:
		return nil, fmt.Errorf("unsupported reference type")
	}
}

func (h *aicallHandler) startReferenceTypeCall(ctx context.Context, c *ai.AI, activeflowID uuid.UUID, referenceID uuid.UUID, gender aicall.Gender, language string) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeCall",
	})

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, cmconfbridge.TypeConference)
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

	res, err := h.UpdateStatusStartProgressing(ctx, tmp.ID, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, errors.Wrapf(err, "Could not update the status to start. aicall_id: %s", tmp.ID)
	}

	return res, nil
}
