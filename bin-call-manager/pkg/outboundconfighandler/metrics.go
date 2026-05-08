package outboundconfighandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

// outboundConfigFetchErrorTotal counts failures of GetByCustomerID, labelled by error type.
//
// Cardinality bound: ≤1 series per error_type. We deliberately avoid a per-customer
// label — at the current scale of ~K customers the cardinality would be acceptable,
// but error_type-only keeps the metric stable as customer count grows. Revisit if
// per-customer visibility becomes necessary.
//
// SLO trigger for adding a circuit-breaker fallback: > 10/min sustained over 5 min.
var outboundConfigFetchErrorTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "call_manager_outbound_config_fetch_error_total",
		Help: "Total OutboundConfig fetch failures by error type.",
	},
	[]string{"error_type"},
)

func init() {
	prometheus.MustRegister(outboundConfigFetchErrorTotal)
}

// IncFetchError increments the fetch-error counter with the given error type.
// Exported so callers outside this package can record failures consistently.
func IncFetchError(errorType string) {
	outboundConfigFetchErrorTotal.WithLabelValues(errorType).Inc()
}
