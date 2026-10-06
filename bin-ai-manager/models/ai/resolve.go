package ai

import "strings"

// RunnerServicePlatformOpenRouter is the internal runner service type for
// platform-managed models. Customers can never select it directly.
const RunnerServicePlatformOpenRouter = "platform_openrouter"

// Outcome is the result class of ResolveEngine.
type Outcome int

const (
	OutcomeRejected Outcome = iota
	OutcomeCatalog
	OutcomeDirectPassthrough
)

// Resolved is what a session start hands over to the runner.
type Resolved struct {
	Entry      *ModelEntry // nil for passthrough/rejected
	RunnerType string      // what the Python runner receives
	BlankKey   bool        // true: never send the customer engine_key
}

var directPassthroughPrefixes = map[string]bool{"openai": true, "gemini": true, "grok": true}

// ResolveEngine maps a customer-facing engine model to the runner type (fail-closed).
func ResolveEngine(m EngineModel) (Resolved, Outcome) {
	for i := range catalog {
		if catalog[i].ID != m || catalog[i].Route == RouteCustomOpenRouter {
			continue
		}

		e := &catalog[i]
		if e.Route == RouteOpenRouter {
			return Resolved{Entry: e, RunnerType: RunnerServicePlatformOpenRouter + "." + e.UpstreamSlug, BlankKey: true}, OutcomeCatalog
		}
		return Resolved{Entry: e, RunnerType: string(m)}, OutcomeCatalog
	}

	parts := strings.SplitN(string(m), ".", 2)
	if len(parts) == 2 && parts[1] != "" && directPassthroughPrefixes[parts[0]] {
		return Resolved{RunnerType: string(m)}, OutcomeDirectPassthrough
	}

	return Resolved{}, OutcomeRejected
}

// IsValidEngineModel keeps its name so existing callers keep compiling; semantics are the catalog policy.
func IsValidEngineModel(m EngineModel) bool {
	_, o := ResolveEngine(m)
	return o != OutcomeRejected
}
