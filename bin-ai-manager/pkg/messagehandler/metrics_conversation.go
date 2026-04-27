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
