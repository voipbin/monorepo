package messagehandler

import "github.com/prometheus/client_golang/prometheus"

var (
	promConversationReplySendTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conversation_reply_send_total",
			Help:      "Total ConversationV1MessageSend attempts from AI delivery, by result (success|failure).",
		},
		[]string{"result"},
	)
	promConversationStaleResponseDroppedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_stale_response_dropped_total",
			Help:      "Stale BotLLM responses dropped by PipecatcallID guard (primary|secondary).",
		},
		[]string{"guard"},
	)
)

func init() {
	prometheus.MustRegister(promConversationReplySendTotal, promConversationStaleResponseDroppedTotal)
}

// Test parallelism note:
// promConversationReplySendTotal and promConversationStaleResponseDroppedTotal
// are package-level Prometheus counters registered in init(). Tests that assert
// counter increments via testutil.ToFloat64 use a before/after snapshot pattern
// (see event_test.go). These tests MUST NOT call t.Parallel() at the test
// function level — concurrent increments would corrupt the delta assertion.
// Sub-tests within a single test function are sequential by default and are
// safe; only top-level parallelism is hazardous here.
//
// If the future test suite needs t.Parallel() on these handlers, the fix is to
// inject an isolated *prometheus.Registry per test rather than sharing the
// global default registry.
