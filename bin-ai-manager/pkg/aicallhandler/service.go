package aicallhandler

import (
	"context"
	"encoding/json"
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
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceStart",
		"ai_id":          aiID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"gender":         gender,
		"language":       language,
	})

	if referenceType == aicall.ReferenceTypeNone {
		return nil, errors.New("unsupported reference type")
	}

	cc, err := h.Start(ctx, aiID, activeflowID, referenceType, referenceID, gender, language)
	if err != nil {
		log.Errorf("Could not start aicall. err: %v", err)
		return nil, fmt.Errorf("could not start aicall. err: %v", err)
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

	res := &commonservice.Service{
		ID:          cc.ID,
		Type:        commonservice.TypeAIcall,
		PushActions: actions,
	}

	return res, nil
}
