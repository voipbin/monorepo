package ai

import "github.com/prometheus/client_golang/prometheus"

// metricsNamespace matches the namespace used by pkg/aicallhandler's
// Prometheus metrics (ai_manager_*) so this counter surfaces alongside the
// rest of the service's metrics.
const metricsNamespace = "ai_manager"

// UnknownAITypeToolDenialTotal counts AllowedToolNames calls that hit the
// deny-by-default branch (an AI Type that is neither TypeNormal nor
// TypeInsight). This should never fire in production -- Create/Update
// resolve TypeNone to TypeNormal before persisting (see pkg/aihandler/
// chatbot.go), so every persisted AI record already has a known Type. A
// non-zero count means either a new Type was added without updating
// AllowedToolNames, or a record reached this path with a corrupted/unknown
// Type value -- either way, tool access was correctly denied rather than
// silently falling back to the widest (Normal, write-capable) set. See
// docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §2.2.
var UnknownAITypeToolDenialTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "unknown_ai_type_tool_denial_total",
		Help:      "Counter of AllowedToolNames resolutions that denied all tools due to an unknown/unhandled AI Type.",
	},
)

func init() {
	prometheus.MustRegister(UnknownAITypeToolDenialTotal)
}
