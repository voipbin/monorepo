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
	// fail-open path where resolveAIFromAIcall fails and the session falls back
	// to GetAll() (all tools, over-broad exposure) instead of the AI's configured
	// tool whitelist. This path is intentionally fail-open (VOIP-1234 §6 v4) since
	// it cannot be scoped to Insight-typed AIs specifically and no incident has
	// been observed to justify a fail-closed change affecting all AICall-backed
	// sessions. This counter (paired with the Errorf log at the call site) exists
	// so a real-world spike is observable and can trigger a design revisit.
	metricsToolResolveFallbackTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pipecat_manager_tool_resolve_fallback_total",
			Help: "Counter of runnerStartScript falling back to GetAll() (all tools) after an AI lookup failure.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		metricsLLMFlushExit,
		metricsIdleWatchdogFired,
		metricsFlushFinalizeOutcome,
		metricsToolResolveFallbackTotal,
	)
}
