package subscribehandler

import "github.com/prometheus/client_golang/prometheus"

// Metrics live package-local (not in a dedicated metricshandler package,
// deviating from docs/conventions/metrics.md §14.1 deliberately): this
// package already registers its batch histograms locally (main.go), and
// eventCh is an unexported field only this package can observe. Same
// shape as bin-schedule-manager/pkg/dispatchhandler/metrics.go.
//
// Naming: the gauge carries a _ratio suffix (instantaneous value), the
// histogram does not (distribution) - they cannot share a base name
// because Prometheus rejects duplicate fully-qualified names at
// MustRegister time regardless of metric type.
var (
	promSubscribeEventDropped = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace, // the existing var in main.go - do not redeclare
		Name:      "subscribe_event_dropped_total",
		Help:      "Events dropped because the in-memory event channel was full. Any nonzero value is permanently lost customer timeline data - see docs/operations.md.",
	})

	promSubscribeEventChannelUsageRatio = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "subscribe_event_channel_usage_ratio",
		Help:      "Instantaneous len/cap of the in-memory event channel (0..1). Scrape-sampled - use subscribe_event_channel_usage (histogram) for burst-accurate percentiles.",
	})

	promSubscribeEventChannelUsage = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricsNamespace,
		Name:      "subscribe_event_channel_usage",
		Help:      "Channel occupancy (len/cap, 0..1) observed at every enqueue attempt. histogram_quantile(0.99, ...) over this answers the 2-replica gate (p99 < 0.5) accurately across sub-scrape bursts, and buckets aggregate across replicas - see subscribe_event_channel_usage_ratio (gauge) for the instantaneous value.",
		Buckets:   []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 0.99, 1},
	})
)

func init() {
	prometheus.MustRegister(
		promSubscribeEventDropped,
		promSubscribeEventChannelUsageRatio,
		promSubscribeEventChannelUsage,
	)
}
