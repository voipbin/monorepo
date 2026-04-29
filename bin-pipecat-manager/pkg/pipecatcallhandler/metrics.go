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
)

func init() {
	prometheus.MustRegister(
		metricsLLMFlushExit,
		metricsIdleWatchdogFired,
		metricsFlushFinalizeOutcome,
	)
}
