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
		variableAIID:          cc.AssistanceID.String(),
		variableAIEngineModel: string(cc.AIEngineModel),
		variableConfbridgeID:  cc.ConfbridgeID.String(),
		variableSTTLanguage:   cc.STTLanguage,
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
		tmp, err := h.substituteText(ctx, activeflowID, a.InitPrompt)
		if err != nil {
			log.Errorf("Could not substitute the init prompt. err: %v", err)
			return res
		} else {
			res = tmp
		}
	}

	return res
}

// substituteText is the single, error-returning flow-variable substitution
// core. It is a plain wrapper over the RPC with NO gating of its own: every
// caller decides for itself whether a given string is worth an RPC, and the
// existing callers (getInitPrompt, getParameterValue) deliberately keep the
// exact call pattern they had before this extraction -- getInitPrompt gates on
// "activeflow set AND prompt non-empty" (no "${" test), getParameterValue calls
// it for every string leaf unconditionally.
func (h *aicallHandler) substituteText(ctx context.Context, activeflowID uuid.UUID, s string) (string, error) {
	res, err := h.reqHandler.FlowV1VariableSubstitute(ctx, activeflowID, s)
	if err != nil {
		return "", errors.Wrap(err, "could not substitute the text")
	}

	return res, nil
}

// substituteValue is the STRICT counterpart of getParameterValue: the same
// recursive map/slice/scalar walk, but the first substitution error aborts the
// whole walk and the partial result is discarded.
//
// It exists for the Insight session refresh only (refreshPrompt), which must
// fail closed rather than persist a half-substituted prompt block. Every other
// caller keeps the lenient per-leaf fallback of getParameterValue.
func (h *aicallHandler) substituteValue(ctx context.Context, activeflowID uuid.UUID, v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		tmp := make(map[string]any)
		for _, key := range rv.MapKeys() {
			k := fmt.Sprintf("%v", key.Interface())
			val, err := h.substituteValue(ctx, activeflowID, rv.MapIndex(key).Interface())
			if err != nil {
				return nil, err
			}
			tmp[k] = val
		}
		return tmp, nil

	case reflect.Slice, reflect.Array:
		tmp := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			val, err := h.substituteValue(ctx, activeflowID, rv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			tmp[i] = val
		}
		return tmp, nil

	case reflect.String:
		return h.substituteText(ctx, activeflowID, rv.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil

	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil

	case reflect.Bool:
		return rv.Bool(), nil
	}

	return v, nil
}

func (h *aicallHandler) getDataAsJSON(ctx context.Context, data map[string]any, activeflowID uuid.UUID) string {
	if data == nil {
		return "{}"
	}

	wg := sync.WaitGroup{}
	tmpRes := sync.Map{}
	for k, v := range data {
		wg.Add(1)

		go func(key string, value any) {
			defer wg.Done()
			data := h.getParameterValue(ctx, value, activeflowID)
			tmpRes.Store(key, data)
		}(k, v)
	}
	wg.Wait()

	tmpMap := map[string]any{}
	tmpRes.Range(func(key, value any) bool {
		k, ok := key.(string)
		if !ok {
			logrus.WithFields(logrus.Fields{
				"func": "getDataAsJSON",
				"key":  key,
			}).Warn("Non-string key encountered in tmpRes; skipping entry")
			return true
		}
		tmpMap[k] = value
		return true
	})

	dataBytes, err := json.Marshal(tmpMap)
	if err != nil {
		logrus.Errorf("Could not marshal data back to string. err: %v", err)
		return "{}"
	}

	return string(dataBytes)
}

func (h *aicallHandler) getParameterValue(ctx context.Context, v any, activeflowID uuid.UUID) any {
	log := logrus.WithFields(logrus.Fields{
		"func":          "getParameterValue",
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
			tmp[k] = h.getParameterValue(ctx, val, activeflowID)
		}
		return tmp

	case reflect.Slice, reflect.Array:
		tmp := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			tmp[i] = h.getParameterValue(ctx, rv.Index(i).Interface(), activeflowID)
		}
		return tmp

	case reflect.String:
		str := v.(string)
		res, err := h.substituteText(ctx, activeflowID, str)
		if err != nil {
			log.Errorf("Could not substitute the parameter string. err: %v", err)
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
