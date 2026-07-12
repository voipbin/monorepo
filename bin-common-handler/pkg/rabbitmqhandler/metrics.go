package rabbitmqhandler

import "github.com/prometheus/client_golang/prometheus"

// Prometheus metrics for the ack-after-process retry mechanism (VOIP-1233).
// Registered via a real Go func init() below (not an explicit call from
// NewRabbit()) so registration safety does not depend on how many times
// NewRabbit() is invoked per process -- package-level init() is guaranteed
// by the Go runtime to run exactly once regardless of caller behavior.
var (
	promEventRetried = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmqhandler_event_retried_total",
			Help: "Count of event messages that failed processing and were scheduled for a delayed retry.",
		},
		[]string{"queue", "retry_count"},
	)

	promEventDropped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmqhandler_event_dropped_total",
			Help: "Count of event messages dropped after exhausting all retries.",
		},
		[]string{"queue"},
	)
)

func init() {
	prometheus.MustRegister(
		promEventRetried,
		promEventDropped,
	)
}
