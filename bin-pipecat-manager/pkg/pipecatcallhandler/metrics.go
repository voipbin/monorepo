package pipecatcallhandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics for the LLM intermediate-flush / finalize subsystem.
//
// Names follow the pipecat_manager_* convention used elsewhere in the service
// (see pkg/listenhandler/main.go). All metrics are registered with the default
// Prometheus registerer in init().
var (
	metricsLLMFlushExit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipecat_manager_llm_flush_exit_total",
			Help: "Counter of runLLMIntermediateFlush goroutine exits, by reason.",
		},
		[]string{"reason"},
	)

	metricsIdleWatchdogFired = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pipecat_manager_llm_idle_watchdog_fired_total",
			Help: "Counter of idle-watchdog firings (no tokens for idleWatchdogTimeout while flushing).",
		},
	)

	metricsFlushFinalizeOutcome = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipecat_manager_llm_flush_finalize_outcome_total",
			Help: "Counter of flushAndFinalize outcomes from the terminate caller's perspective.",
		},
		[]string{"outcome"},
	)

	// metricsToolResolveFallbackTotal counts occurrences of the runnerStartScript
	// fail-CLOSED path where resolveAIFromAIcall fails and the session falls
	// back to an empty tool list instead of the AI's configured tool
	// whitelist. This was originally a fail-OPEN path (falling back to
	// GetAll(), VOIP-1234 §6 v4); that decision was reversed (design
	// docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md
	// §2.4) because tool-access control is a case where least-privilege must
	// outweigh availability -- a degraded (tool-less) session is acceptable,
	// silently granting write-capable tools to a session whose type couldn't
	// be determined is not. This counter (paired with the Errorf log at the
	// call site) exists so a real-world spike in this path is observable.
	metricsToolResolveFallbackTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pipecat_manager_tool_resolve_fallback_total",
			Help: "Counter of runnerStartScript failing closed to an empty tool list after an AI lookup failure.",
		},
	)

	// metricsMcpToolListFallbackTotal counts occurrences of the
	// AIV1AIcallToolList callback RPC failing inside runnerStartScript
	// (design docs/plans/2026-09-29-mcp-tool-exposure-pr-b2-design.md §2.2,
	// §9, D13). Unlike metricsToolResolveFallbackTotal above (which fails
	// CLOSED to an empty tool list on AI-resolution failure), this counter
	// tracks a narrower, later failure: the AI was resolved and its
	// built-in tools are already in hand, only the MCP-tool callback
	// itself failed. Per D13 the session fails OPEN here -- it continues
	// with the built-in tools it already has rather than losing the whole
	// tool list. The failure reason is captured in the accompanying
	// Warnf log line, not as a metric label (cardinality discipline, §9).
	metricsMcpToolListFallbackTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pipecat_manager_mcp_tool_list_fallback_total",
			Help: "Counter of runnerStartScript falling back to built-in tools only after the AIV1AIcallToolList callback RPC failed.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		metricsLLMFlushExit,
		metricsIdleWatchdogFired,
		metricsFlushFinalizeOutcome,
		metricsToolResolveFallbackTotal,
		metricsMcpToolListFallbackTotal,
	)
}
