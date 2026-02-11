package metricshandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

const metricsNamespace = "agent_manager"

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

	// ReceivedSubscribeEventProcessTime tracks subscribe event processing time.
	ReceivedSubscribeEventProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_subscribe_event_process_time",
			Help:      "Process time of received subscribe event",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"publisher", "type"},
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

	// LoginTotal counts agent login attempts by outcome.
	LoginTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "login_total",
			Help:      "Total number of agent login attempts",
		},
		[]string{"status"},
	)

	// PasswordResetTotal counts password reset attempts by outcome.
	PasswordResetTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "password_reset_total",
			Help:      "Total number of password reset attempts",
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(
		ReceivedRequestProcessTime,
		ReceivedSubscribeEventProcessTime,
		DBOperationDuration,
		DBOperationTotal,
		CacheOperationTotal,
		RPCCallDuration,
		RPCCallTotal,
		LoginTotal,
		PasswordResetTotal,
	)
}
