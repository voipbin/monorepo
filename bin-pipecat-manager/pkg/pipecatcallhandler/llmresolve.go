package pipecatcallhandler

import (
	"fmt"

	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// resolveSessionLLM maps the customer-facing model to what the runner receives.
// aiKey is the customer's engine_key ("" when unknown).
//
// It resolves from the model id alone (no RPC), so a failed AI lookup cannot
// bypass it. Fail-closed: a rejected model returns an error. Platform-routed
// (OpenRouter) models get the internal runner type and a blank key, so the
// customer's engine_key is never forwarded for them.
func resolveSessionLLM(llmType pipecatcall.LLMType, aiKey string) (runnerType string, runnerKey string, err error) {
	r, outcome := amai.ResolveEngine(amai.EngineModel(llmType))
	if outcome == amai.OutcomeRejected {
		return "", "", fmt.Errorf("engine model is not available: %s", llmType)
	}

	if r.BlankKey {
		return r.RunnerType, "", nil
	}

	return r.RunnerType, aiKey, nil
}
