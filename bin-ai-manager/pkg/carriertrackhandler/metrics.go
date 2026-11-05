package carriertrackhandler

import "github.com/prometheus/client_golang/prometheus"

const (
	metricsNamespace = "ai_manager"

	// outcomeSuccess and outcomeCacheHit are the outcomes that are not error codes.
	outcomeSuccess  = "success"
	outcomeCacheHit = "cache_hit"
)

// The labels are a fixed set: the carrier name and the outcome. A customer,
// server, tracking number or key is never a label.
var (
	promLookupTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "carrier_lookup_total",
			Help:      "Total number of carrier tracking lookups by carrier and outcome (success, cache_hit or an error code).",
		},
		[]string{"carrier", "outcome"},
	)
	promLookupDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "carrier_lookup_duration_seconds",
			Help:      "Duration of carrier tracking lookups that reached the carrier, by carrier.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2, 4, 6, 8, 10},
		},
		[]string{"carrier"},
	)
	promInflight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "carrier_inflight",
			Help:      "Number of carrier tracking lookups running now, by carrier.",
		},
		[]string{"carrier"},
	)
	promTokenExchangeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "carrier_token_exchange_total",
			Help:      "Total number of carrier access token exchanges by carrier and outcome.",
		},
		[]string{"carrier", "outcome"},
	)
)

func init() {
	prometheus.MustRegister(
		promLookupTotal,
		promLookupDurationSeconds,
		promInflight,
		promTokenExchangeTotal,
	)
}
