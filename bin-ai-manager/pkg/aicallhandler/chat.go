package aicallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
)

func (h *aicallHandler) setActiveflowVariables(ctx context.Context, cc *aicall.AIcall) error {
	if cc.ActiveflowID == uuid.Nil {
		// nothing todo
		return nil
	}

	variables := map[string]string{
		variableID:            cc.ID.String(),
		variableAIID:          cc.AIID.String(),
		variableAIEngineModel: string(cc.AIEngineModel),
		variableConfbridgeID:  cc.ConfbridgeID.String(),
		variableGender:        string(cc.Gender),
		variableLanguage:      cc.Language,
		variablePipecatcallID: cc.PipecatcallID.String(),
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, cc.ActiveflowID, variables); errSet != nil {
		return errors.Wrap(errSet, "could not set the variables")
	}
	return nil
}

func (h *aicallHandler) getInitPrompt(ctx context.Context, a *ai.AI, activeflowID uuid.UUID) string {
	log := logrus.WithFields(logrus.Fields{
		"func":          "chatGetInitPrompt",
		"ai_id":         a.ID,
		"activeflow_id": activeflowID,
	})

	res := a.InitPrompt
	if activeflowID != uuid.Nil {
		tmp, err := h.reqHandler.FlowV1VariableSubstitute(ctx, activeflowID, a.InitPrompt)
		if err != nil {
			log.Errorf("Could not substitute the init prompt. err: %v", err)
			return res
		} else {
			res = tmp
		}
	}

	return res
}
