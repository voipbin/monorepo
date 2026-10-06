package pipecatcallhandler

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// errEngineModelNotAvailable builds the rejection error. A value with the custom
// prefix is customer free text (it may even be a pasted key), so it is never
// echoed back or logged.
func errEngineModelNotAvailable(llmType pipecatcall.LLMType) error {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(llmType))), amai.EngineModelPrefixCustomOpenRouter) {
		return errors.New("engine model is not available")
	}
	return fmt.Errorf("engine model is not available: %s", llmType)
}

// classifySessionLLM only decides whether the model is selectable. It never looks
// at a key, so the keyless pre-check in Start() keeps working for custom models.
func classifySessionLLM(llmType pipecatcall.LLMType) error {
	if _, outcome := amai.ResolveEngine(amai.EngineModel(llmType)); outcome == amai.OutcomeRejected {
		return errEngineModelNotAvailable(llmType)
	}

	return nil
}

// resolveSessionLLM maps the customer-facing model to what the runner receives.
// aiKey is the customer's engine_key ("" when unknown).
//
// It resolves from the model id alone (no RPC), so a failed AI lookup cannot
// bypass it. Fail-closed: a rejected model returns an error. Platform-routed
// (OpenRouter) models get the internal runner type and a blank key, so the
// customer's engine_key is never forwarded for them. Custom OpenRouter models
// require the customer's key: an empty key fails and only the trimmed key is
// forwarded.
func resolveSessionLLM(llmType pipecatcall.LLMType, aiKey string) (runnerType string, runnerKey string, err error) {
	r, outcome := amai.ResolveEngine(amai.EngineModel(llmType))
	if outcome == amai.OutcomeRejected {
		return "", "", errEngineModelNotAvailable(llmType)
	}

	if r.BlankKey {
		return r.RunnerType, "", nil
	}

	if r.RequireKey {
		key := strings.TrimSpace(aiKey)
		if key == "" {
			return "", "", errors.New("custom engine key is empty")
		}
		return r.RunnerType, key, nil
	}

	return r.RunnerType, aiKey, nil
}
