package ai

import (
	"errors"
	"regexp"
	"strings"
)

// RunnerServicePlatformOpenRouter is the internal runner service type for
// platform-managed models. Customers can never select it directly.
const RunnerServicePlatformOpenRouter = "platform_openrouter"

// Outcome is the result class of ResolveEngine.
type Outcome int

const (
	OutcomeRejected Outcome = iota
	OutcomeCatalog
	OutcomeDirectPassthrough
	OutcomeCustomOpenRouter // appended last to keep the iota values stable
)

// Resolved is what a session start hands over to the runner.
type Resolved struct {
	Entry      *ModelEntry // nil for passthrough/rejected
	RunnerType string      // what the Python runner receives
	BlankKey   bool        // true: never send the customer engine_key
	RequireKey bool        // custom models: an empty key is a hard failure
}

var (
	ErrInvalidEngineModel = errors.New("invalid engine model")
	ErrEngineKeyRequired  = errors.New("engine key required")
)

// customModelIDPattern is the "<author>/<slug>" form: exactly one slash, no variant suffix.
var customModelIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)

// maxEngineModelLen is the engine_model column width.
const maxEngineModelLen = 255

// ValidateCustomModelID checks the part after the "openrouter." prefix.
// The total length with the prefix must stay within the engine_model column.
func ValidateCustomModelID(id string) bool {
	if len(id) == 0 || len(id) > maxEngineModelLen-len(EngineModelPrefixCustomOpenRouter) {
		return false
	}
	if !customModelIDPattern.MatchString(id) {
		return false
	}

	// "openrouter/auto" and similar router IDs are not models.
	author := strings.SplitN(id, "/", 2)[0]
	return !strings.EqualFold(author, "openrouter")
}

// ValidateEngine checks the final (model, key) state of an AI.
// modelChanged is false when an update keeps the stored model, so an unchanged legacy value keeps saving.
func ValidateEngine(model EngineModel, key string, modelChanged bool) error {
	if modelChanged {
		if _, o := ResolveEngine(model); o == OutcomeRejected {
			return ErrInvalidEngineModel
		}
	}

	if strings.HasPrefix(string(model), EngineModelPrefixCustomOpenRouter) && strings.TrimSpace(key) == "" {
		return ErrEngineKeyRequired
	}

	return nil
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

	// Custom OpenRouter: exact, case-sensitive prefix. The runner type keeps the "openrouter."
	// prefix and never becomes platform_openrouter, so the platform key stays unreachable.
	if strings.HasPrefix(string(m), EngineModelPrefixCustomOpenRouter) {
		if ValidateCustomModelID(strings.TrimPrefix(string(m), EngineModelPrefixCustomOpenRouter)) {
			return Resolved{RunnerType: string(m), RequireKey: true}, OutcomeCustomOpenRouter
		}

		return Resolved{}, OutcomeRejected
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
