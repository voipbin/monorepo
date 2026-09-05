package messagehandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	promForeignPipecatcallDroppedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_foreign_pipecatcall_dropped_total",
			Help:      "Total number of pipecat message events dropped because they came from a pipecatcall the AIcall no longer considers its conversational turn. Covers both Insight listen-turn output and pre-existing stale contact_case replies, which used to be persisted silently.",
		},
		[]string{"handler"},
	)
)

func init() {
	prometheus.MustRegister(
		promForeignPipecatcallDroppedTotal,
	)
}
