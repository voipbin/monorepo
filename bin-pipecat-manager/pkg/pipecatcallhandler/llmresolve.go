package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// errEngineModelNotAvailable builds the rejection error. A value with the custom
// prefix is customer free text (it may even be a pasted key), so it is never
// echoed back or logged.
func errEngineModelNotAvailable(llmType pipecatcall.LLMType) error {
	lower := strings.ToLower(strings.TrimSpace(string(llmType)))
	if strings.HasPrefix(lower, amai.EngineModelPrefixCustomOpenRouter) || strings.HasPrefix(lower, amai.CustomEndpointPrefix) {
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

// engineVendor returns the lower-cased vendor prefix of an engine model ("openai" for
// "openai.gpt-5", "openrouter" for "openrouter.vendor/model"). It identifies which
// provider a customer key belongs to.
func engineVendor(m string) string {
	return strings.ToLower(strings.SplitN(strings.TrimSpace(m), ".", 2)[0])
}

// checkLiveEngineVendor guards the customer key against a stale session model.
//
// The session llm_type is a snapshot taken when the aicall was created, while the
// key is read live from the AI on every start. If the AI was edited to another
// vendor in between (for example from a custom OpenRouter model to an OpenAI model),
// the new key would be sent to the old vendor. When a key is about to be forwarded
// and the live AI's vendor differs from the session's, the start is rejected.
// A model change within the same vendor keeps working. liveAI is nil when the AI
// lookup failed; then no live key was read, so there is nothing to protect.
func checkLiveEngineVendor(llmType pipecatcall.LLMType, liveAI *amai.AI, runnerKey string) error {
	if liveAI == nil || runnerKey == "" {
		return nil
	}

	if engineVendor(string(llmType)) != engineVendor(string(liveAI.EngineModel)) {
		return errors.New("engine model of the ai has changed since the session was created")
	}

	return nil
}

// endpointLookupTimeout bounds the start path lookup of a custom endpoint host.
const endpointLookupTimeout = time.Second

// errEngineEndpointNotAvailable is the one fixed error of every custom endpoint start failure. It
// carries no URL, key or cause (the URL may hold what the customer pasted).
var errEngineEndpointNotAvailable = errors.New("engine endpoint is not available")

// sessionEndpoint is the customer specified endpoint of a session, read from the live AI.
type sessionEndpoint struct {
	BaseURL amai.RedactedString
	BaseKey amai.RedactedString
}

// resolveSessionEndpoint returns the endpoint the runner must use. It returns nil unless the
// session's own model resolves to the custom endpoint outcome, so a URL is never forwarded for
// another type even if the live AI changed. For a custom session it fails closed when the live AI
// is missing, belongs to another customer, is deleted, has no URL, has a stored key it could not
// decrypt, or when the host does not pass the network validation.
func resolveSessionEndpoint(ctx context.Context, llmType pipecatcall.LLMType, customerID uuid.UUID, liveAI *amai.AI) (*sessionEndpoint, error) {
	if _, outcome := amai.ResolveEngine(amai.EngineModel(llmType)); outcome != amai.OutcomeCustomEndpoint {
		return nil, nil
	}

	if liveAI == nil || liveAI.CustomerID != customerID || liveAI.TMDelete != nil || liveAI.EngineBaseURL == "" {
		return nil, errEngineEndpointNotAvailable
	}
	if len(liveAI.EngineBaseKeyEnc) > 0 && liveAI.EngineBaseKey == "" {
		return nil, errEngineEndpointNotAvailable
	}

	cctx, cancel := context.WithTimeout(ctx, endpointLookupTimeout)
	defer cancel()
	if err := amai.ValidateEngineBaseURL(cctx, liveAI.EngineBaseURL); err != nil {
		return nil, errEngineEndpointNotAvailable
	}

	return &sessionEndpoint{BaseURL: amai.RedactedString(liveAI.EngineBaseURL), BaseKey: liveAI.EngineBaseKey}, nil
}
