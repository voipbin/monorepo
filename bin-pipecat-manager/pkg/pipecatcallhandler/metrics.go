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

	// metricsPipelineErrorTotal counts every RTVI "error" frame received from the pipecat runner,
	// by classified category and fatal flag (VOIP-1542). Counted whether or not a customer notice
	// is published, so operators see the full error rate.
	metricsPipelineErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipecat_manager_pipeline_error_total",
			Help: "Counter of RTVI error frames received from the pipecat runner, by category and fatal flag.",
		},
		[]string{"category", "fatal"},
	)

	// metricsRTVIErrorResponseTotal counts RTVI "error-response" frames: the runner rejected a
	// client request pipecat-manager sent (e.g. send-text). A platform contract failure, kept
	// separate from provider errors so it does not pollute pipeline-error alerting (VOIP-1542).
	metricsRTVIErrorResponseTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pipecat_manager_rtvi_error_response_total",
			Help: "Counter of RTVI error-response frames (runner rejected a request from pipecat-manager).",
		},
	)
)

func init() {
	prometheus.MustRegister(
		metricsLLMFlushExit,
		metricsIdleWatchdogFired,
		metricsFlushFinalizeOutcome,
		metricsToolResolveFallbackTotal,
		metricsPipelineErrorTotal,
		metricsRTVIErrorResponseTotal,
	)
}
