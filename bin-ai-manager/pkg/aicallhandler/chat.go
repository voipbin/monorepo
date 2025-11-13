package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	reflect "reflect"

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
	if activeflowID != uuid.Nil && a.InitPrompt != "" {
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

func (h *aicallHandler) getEngineData(ctx context.Context, a *ai.AI, activeflowID uuid.UUID) string {
	log := logrus.WithFields(logrus.Fields{
		"func":          "getEngineData",
		"ai_id":         a.ID,
		"activeflow_id": activeflowID,
	})

	tmpRes := map[string]string{}
	for k, v := range a.EngineData {
		val := h.getEngineDataString(ctx, v, activeflowID)
		tmpRes[k] = val
	}

	// marshal back to string
	engineDataBytes, err := json.Marshal(tmpRes)
	if err != nil {
		log.Errorf("Could not marshal the engine data back to string. err: %v", err)
		return ""
	}

	return string(engineDataBytes)
}

func (h *aicallHandler) getEngineDataString(ctx context.Context, v any, activeflowID uuid.UUID) string {
	log := logrus.WithFields(logrus.Fields{
		"func":          "getEngineDataString",
		"activeflow_id": activeflowID,
	})

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		tmp := make(map[string]string)
		for _, key := range rv.MapKeys() {
			k := fmt.Sprintf("%v", key.Interface())
			val := rv.MapIndex(key).Interface()
			tmp[k] = h.getEngineDataString(ctx, val, activeflowID)
		}

		tmpRes, err := json.Marshal(tmp)
		if err != nil {
			log.Errorf("Could not marshal map. err: %v", err)
			return ""
		}

		res := string(tmpRes)
		return res

	case reflect.String:
		v := v.(string)

		res, err := h.reqHandler.FlowV1VariableSubstitute(ctx, activeflowID, v)
		if err != nil {
			log.Errorf("Could not substitute the engine data string. err: %v", err)
			return v
		}

		return res

	case reflect.Slice, reflect.Array:
		tmp := make([]string, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			tmp[i] = h.getEngineDataString(ctx, rv.Index(i).Interface(), activeflowID)
		}
		tmpRes, err := json.Marshal(tmp)
		if err != nil {
			log.Errorf("Could not marshal slice. err: %v", err)
			return ""
		}

		res := string(tmpRes)
		return res

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return fmt.Sprintf("%v", v)
	}

	return ""
}
