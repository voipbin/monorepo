package aicallhandler

import (
	"context"
	"fmt"

	commonservice "monorepo/bin-common-handler/models/service"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
)

// ServiceStart is start a new aicall service.
func (h *aicallHandler) ServiceStart(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*commonservice.Service, error) {

	switch referenceType {
	case aicall.ReferenceTypeCall:
		return h.serviceStartReferenceTypeCall(ctx, aiID, activeflowID, referenceID, gender, language)

	case aicall.ReferenceTypeConversation:
		return h.serviceStartReferenceTypeConversation(ctx, aiID, activeflowID, referenceID, gender, language)

	default:
		return nil, fmt.Errorf("unsupported reference type. reference_type: %s", referenceType)
	}
}

// serviceStartReferenceTypeCall is start a new aicall for call reference type.
func (h *aicallHandler) serviceStartReferenceTypeCall(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "serviceStartReferenceTypeCall",
		"ai_id":         aiID,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
		"gender":        gender,
		"language":      language,
	})

	cc, err := h.Start(ctx, aiID, activeflowID, aicall.ReferenceTypeCall, referenceID, gender, language)
	if err != nil {
		log.Errorf("Could not start aicall. err: %v", err)
		return nil, fmt.Errorf("could not start aicall. err: %v", err)
	}
	log.WithField("aicall", cc).Debugf("Started aicall. aicall_id: %s", cc.ID)

	actions := []fmaction.Action{
		{
			ID:   h.utilHandler.UUIDCreate(),
			Type: fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
				ConfbridgeID: cc.ConfbridgeID,
			}),
		},
	}

	res := &commonservice.Service{
		ID:          cc.ID,
		Type:        commonservice.TypeAIcall,
		PushActions: actions,
	}

	return res, nil
}

// serviceStartReferenceTypeConversation is start a new aicall for conversation reference type.
func (h *aicallHandler) serviceStartReferenceTypeConversation(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "serviceStartReferenceTypeConversation",
		"ai_id":         aiID,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
		"gender":        gender,
		"language":      language,
	})

	cc, err := h.Start(ctx, aiID, activeflowID, aicall.ReferenceTypeConversation, referenceID, gender, language)
	if err != nil {
		log.Errorf("Could not start aicall. err: %v", err)
		return nil, fmt.Errorf("could not start aicall. err: %v", err)
	}
	log.WithField("aicall", cc).Debugf("Started aicall. aicall_id: %s", cc.ID)

	res := &commonservice.Service{
		ID:          cc.ID,
		Type:        commonservice.TypeAIcall,
		PushActions: []fmaction.Action{},
	}

	return res, nil
}

func (h *aicallHandler) ServiceStartTypeTask(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ServiceStartTypeTask",
		"ai_id":         aiID,
		"activeflow_id": activeflowID,
	})

	cc, err := h.StartTask(ctx, aiID, activeflowID)
	if err != nil {
		log.Errorf("Could not start aicall. err: %v", err)
		return nil, fmt.Errorf("could not start aicall. err: %v", err)
	}
	log.WithField("aicall", cc).Debugf("Started aicall. aicall_id: %s", cc.ID)

	res := &commonservice.Service{
		ID:   cc.ID,
		Type: commonservice.TypeAIcall,
		PushActions: []fmaction.Action{
			{
				ID:     h.utilHandler.UUIDCreate(),
				Type:   fmaction.TypeBlock,
				Option: fmaction.ConvertOption(fmaction.OptionBlock{}),
			},
		},
	}

	return res, nil
}

func (h *aicallHandler) serviceStop(ctx context.Context, c *aicall.AIcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "serviceStop",
		"aicall_id": c.ID,
	})
	log.Debugf("Stopping the aicall service.")

	tmp, err := h.ProcessTerminate(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("could not start terminating the aicall. err: %v", err)
	}
	log.WithField("aicall", tmp).Debugf("Stopped the aicall service successfully.")

	return nil
}
