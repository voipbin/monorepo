package metricshandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

const metricsNamespace = "customer_manager"

var (
	// ReceivedRequestProcessTime tracks RPC request processing time.
	ReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"type", "method"},
	)

	// DBOperationDuration tracks database operation latency in milliseconds.
	DBOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "db_operation_duration",
			Help:      "Duration of database operations in milliseconds",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"operation", "entity"},
	)

	// DBOperationTotal counts database operations by outcome.
	DBOperationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "db_operation_total",
			Help:      "Total number of database operations",
		},
		[]string{"operation", "entity", "status"},
	)

	// CacheOperationTotal counts cache operations by result (hit/miss).
	CacheOperationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "cache_operation_total",
			Help:      "Total number of cache operations",
		},
		[]string{"operation", "entity", "result"},
	)

	// EventPublishTotal counts published events by type.
	EventPublishTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "event_publish_total",
			Help:      "Total number of published events",
		},
		[]string{"type"},
	)

	// RPCCallDuration tracks cross-service RPC call latency in milliseconds.
	RPCCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "rpc_call_duration",
			Help:      "Duration of cross-service RPC calls in milliseconds",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"service", "method"},
	)

	// RPCCallTotal counts cross-service RPC calls by outcome.
	RPCCallTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "rpc_call_total",
			Help:      "Total number of cross-service RPC calls",
		},
		[]string{"service", "method", "status"},
	)

	// SignupTotal counts customer signup attempts by outcome.
	SignupTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "signup_total",
			Help:      "Total number of customer signup attempts",
		},
		[]string{"status"},
	)

	// EmailVerificationTotal counts email verification attempts by outcome.
	EmailVerificationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "email_verification_total",
			Help:      "Total number of email verification attempts",
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(
		ReceivedRequestProcessTime,
		DBOperationDuration,
		DBOperationTotal,
		CacheOperationTotal,
		EventPublishTotal,
		RPCCallDuration,
		RPCCallTotal,
		SignupTotal,
		EmailVerificationTotal,
	)
}
