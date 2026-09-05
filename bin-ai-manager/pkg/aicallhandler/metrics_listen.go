package aicallhandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics for the Insight AI's realtime call listening. The namespace is
// prepended by the Prometheus client library from metricsNamespace in main.go,
// so Name values here are bare -- writing "ai_manager_" into the name string
// would render as ai_manager_ai_manager_...
var (
	promListenMembershipCheckFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_membership_check_failed_total",
			Help:      "Total number of listen-turn membership checks that errored and degraded to treating the tool call as a real Q&A turn. Near-zero expected; a sustained non-zero rate means Redis is unhealthy, not that anything listen-specific is wrong.",
		},
	)

	promListenStartTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_start_total",
			Help:      "Total number of listen-start attempts by kind and outcome. kind: call, conversation, or unknown for the gates that run before the Case's reference type is known. result: started, reused, skipped_not_listenable, skipped_confbridge_not_ready, skipped_confbridge_error, skipped_start_locked, failed.",
		},
		[]string{"kind", "result"},
	)

	promListenTurnTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_turn_total",
			Help:      "Total listen evaluation turns by kind and outcome. skipped_locked measured against ran is the direct read on how much LLM spend the debounce is saving -- near-zero skipped_locked means the interval is too short for the traffic. skipped_case_closed is the conversation kind's stop signal.",
		},
		[]string{"kind", "result"},
	)

	promListenSegmentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_segment_total",
			Help:      "Total transcript segments seen by listen intake, by outcome. dropped_unknown dominates by design -- this handler sees every final STT result platform-wide.",
		},
		[]string{"result"},
	)

	promListenConversationSegmentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_conversation_segment_total",
			Help:      "Total conversation messages seen by listen intake, by outcome: buffered, dropped_deleted, dropped_empty, dropped_unknown (no listener resolved, or the resolver errored), dropped_stale (a resolved AIcall is already over, or its pointer names another conversation), dropped_tenant_mismatch, failed. dropped_unknown dominates by design -- this handler sees every conversation message platform-wide; dropped_tenant_mismatch must stay at zero.",
		},
		[]string{"result"},
	)

	promListenConversationFlushTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_conversation_flush_total",
			Help:      "Deferred flush timers for conversation listening, by outcome: ran (won the lock and invoked a turn; read against turn_total skipped_empty), skipped_locked, skipped_scheduled (a timer was already armed for this AIcall on this replica).",
		},
		[]string{"result"},
	)

	promListenStopFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_stop_failed_total",
			Help:      "Total listen transcribe-stop RPCs that failed and fell back to the call-hangup-ends-the-audio-transport backstop.",
		},
	)

	promListenNotifyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_notify_total",
			Help:      "Total number of proactive notifications actually delivered to an agent's Insight panel, by listen kind.",
		},
		[]string{"kind"},
	)
)

func init() {
	prometheus.MustRegister(
		promListenMembershipCheckFailedTotal,
		promListenNotifyTotal,
		promListenStartTotal,
		promListenTurnTotal,
		promListenSegmentTotal,
		promListenConversationSegmentTotal,
		promListenConversationFlushTotal,
		promListenStopFailedTotal,
	)
}
