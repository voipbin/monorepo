package emailhandler

import (
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/prometheus/client_golang/prometheus"
)

// metricsNamespace is not declared elsewhere in this package (confirmed via
// codebase search — the only pre-existing metricsNamespace in this service
// lives in pkg/listenhandler/main.go, a different package), so it is defined
// here for emailhandler's own metrics. VOIP-1259.
var metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameEmailManager)

// promOutboundRateLimitedTotal counts outbound sends rejected by the
// per-customer rate limit, by resource_type and result. VOIP-1259.
// Only the "rejected" result is incremented today (the "allowed" side is
// intentionally not tracked per-email — would be extremely high-cardinality/
// high-volume with little operational value), consistent with
// promAIcallContactCaseRecreateRateLimitedTotal which also only counts the
// blocked case.
var promOutboundRateLimitedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "outbound_rate_limited_total",
		Help:      "Total outbound sends rejected by the per-customer rate limit, by resource_type and result (VOIP-1259).",
	},
	[]string{"resource_type", "result"},
)

func init() {
	prometheus.MustRegister(promOutboundRateLimitedTotal)
}
