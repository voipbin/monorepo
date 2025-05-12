package aicallhandler

import (
	"context"
	"fmt"

	commonservice "monorepo/bin-common-handler/models/service"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
)

// ServiceStart is creating a new aicall.
// it increases corresponded counter
func (h *aicallHandler) ServiceStart(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
	resume bool,
) (*commonservice.Service, error) {
	if referenceType == aicall.ReferenceTypeNone {
		return nil, errors.New("unsupported reference type")
	}

	switch referenceType {
	case aicall.ReferenceTypeCall:
		return h.serviceStartReferenceTypeCall(
			ctx,
			aiID,
			activeflowID,
			referenceID,
			gender,
			language,
			resume,
		)

	case aicall.ReferenceTypeConversation:
		return h.serviceStartReferenceTypeConversation(
			ctx,
			aiID,
			activeflowID,
			referenceID,
			gender,
			language,
			resume,
		)

	default:
		return nil, errors.New("unsupported reference type")
	}
}

// serviceStartReferenceTypeCall is creating a new aicall.
// it increases corresponded counter
func (h *aicallHandler) serviceStartReferenceTypeCall(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
	resume bool,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "serviceStartReferenceTypeCall",
		"ai_id":         aiID,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
		"gender":        gender,
		"language":      language,
	})

	cc, err := h.Start(ctx, aiID, activeflowID, aicall.ReferenceTypeCall, referenceID, gender, language, resume)
	if err != nil {
		log.Errorf("Could not start aicall. err: %v", err)
		return nil, fmt.Errorf("could not start aicall. err: %v", err)
	}

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

// serviceStartReferenceTypeConversation is creating a new aicall.
// it increases corresponded counter
func (h *aicallHandler) serviceStartReferenceTypeConversation(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
	resume bool,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "serviceStartReferenceTypeConversation",
		"ai_id":         aiID,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
		"gender":        gender,
		"language":      language,
	})

	cc, err := h.Start(ctx, aiID, activeflowID, aicall.ReferenceTypeConversation, referenceID, gender, language, resume)
	if err != nil {
		log.Errorf("Could not start aicall. err: %v", err)
		return nil, fmt.Errorf("could not start aicall. err: %v", err)
	}

	res := &commonservice.Service{
		ID:          cc.ID,
		Type:        commonservice.TypeAIcall,
		PushActions: []fmaction.Action{},
	}

	return res, nil
}
