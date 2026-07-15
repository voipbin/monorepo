package aicallhandler

import "github.com/prometheus/client_golang/prometheus"

var promAIcallToolCallSessionCapExceededTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "aicall_tool_call_session_cap_exceeded_total",
		Help:      "Total tool calls rejected because the AIcall session tool-call cap was exceeded (VOIP-1259).",
	},
)

func init() {
	prometheus.MustRegister(promAIcallToolCallSessionCapExceededTotal)
}
