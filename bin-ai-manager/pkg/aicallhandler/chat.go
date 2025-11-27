package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	reflect "reflect"
	"sync"

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

	if a.EngineData == nil {
		return "{}"
	}

	wg := sync.WaitGroup{}
	tmpRes := sync.Map{}
	for k, v := range a.EngineData {
		wg.Add(1)

		go func(key string, value any) {
			defer wg.Done()

			// EngineData(value) must be immutable. Concurrent read is safe, but no mutation is allowed after read begins.
			data := h.getEngineDataValue(ctx, value, activeflowID)
			tmpRes.Store(key, data)
		}(k, v)
	}
	wg.Wait()

	tmpMap := map[string]any{}
	tmpRes.Range(func(key, value any) bool {
		k, ok := key.(string)
		if !ok {
			logrus.WithFields(logrus.Fields{
				"func": "getEngineData",
				"key":  key,
			}).Warn("Non-string key encountered in tmpRes; skipping entry")
			return true
		}
		tmpMap[k] = value

		return true
	})

	engineDataBytes, err := json.Marshal(tmpMap)
	if err != nil {
		log.Errorf("Could not marshal the engine data back to string. err: %v", err)
		return "{}"
	}

	return string(engineDataBytes)
}

func (h *aicallHandler) getEngineDataValue(ctx context.Context, v any, activeflowID uuid.UUID) any {
	log := logrus.WithFields(logrus.Fields{
		"func":          "getEngineDataValue",
		"activeflow_id": activeflowID,
	})

	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		tmp := make(map[string]any)
		for _, key := range rv.MapKeys() {
			k := fmt.Sprintf("%v", key.Interface())
			val := rv.MapIndex(key).Interface()
			tmp[k] = h.getEngineDataValue(ctx, val, activeflowID)
		}
		return tmp

	case reflect.Slice, reflect.Array:
		tmp := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			tmp[i] = h.getEngineDataValue(ctx, rv.Index(i).Interface(), activeflowID)
		}
		return tmp

	case reflect.String:
		str := v.(string)
		res, err := h.reqHandler.FlowV1VariableSubstitute(ctx, activeflowID, str)
		if err != nil {
			log.Errorf("Could not substitute the engine data string. err: %v", err)
			return str
		}
		return res

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()

	case reflect.Float32, reflect.Float64:
		return rv.Float()

	case reflect.Bool:
		return rv.Bool()
	}

	return v
}
