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
			Help:      "Total number of listen-start attempts by outcome. result values: started, reused, skipped_not_listenable, skipped_confbridge_not_ready, skipped_confbridge_error, skipped_start_locked, failed. All are values of the existing 'result' label -- no new CounterVec.",
		},
		[]string{"result"},
	)

	promListenNotifyTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_notify_total",
			Help:      "Total number of proactive notifications actually delivered to an agent's Insight panel.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promListenMembershipCheckFailedTotal,
		promListenNotifyTotal,
		promListenStartTotal,
	)
}
