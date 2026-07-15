package messagehandler

import "github.com/prometheus/client_golang/prometheus"

// metricsNamespace is already declared in main.go:64 — do NOT redeclare here.

// promOutboundRateLimitedTotal counts outbound sends rejected by the
// per-customer rate limit, by resource_type and result. VOIP-1259.
// Only the "rejected" result is incremented today (the "allowed" side is
// intentionally not tracked per-message — would be extremely high-cardinality/
// high-volume with little operational value), consistent with
// bin-call-manager's promOutboundRateLimitedTotal.
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
