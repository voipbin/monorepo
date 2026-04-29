package aicallhandler

import "github.com/prometheus/client_golang/prometheus"

var (
	promBackstopReplyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_backstop_reply_total",
			Help:      "Counter of pipecatcall_terminated backstop attempts in messagehandler.EventPMPipecatcallTerminated.",
		},
		[]string{"result"},
	)
)

func init() {
	prometheus.MustRegister(promBackstopReplyTotal)
}
