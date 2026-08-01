package dispatchhandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

const promNamespace = "schedule_manager"

// dispatch result label values (design §5.7). Success/failed/abandoned reuse
// the execution status strings.
const (
	resultSkippedLock    = "skipped_lock"
	resultSkippedCAS     = "skipped_cas"
	resultSkippedOverlap = "skipped_overlap"
)

var (
	promDispatchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "dispatch_total",
			Help:      "Total number of dispatch outcomes per schedule. The abandoned result is counted from the bulk reap without a schedule_name label (empty label).",
		},
		[]string{"schedule_name", "result"},
	)

	promDispatchDurationMS = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: promNamespace,
			Name:      "dispatch_duration_ms",
			Help:      "Dispatch wall time in milliseconds per schedule.",
			Buckets:   []float64{50, 100, 500, 1000, 3000, 10000, 60000, 300000},
		},
		[]string{"schedule_name"},
	)

	promScheduleLastSuccessTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "schedule_last_success_timestamp",
			Help:      "Unix timestamp of the schedule's last successful dispatch. Aggregate with max by (schedule_name) across replicas.",
		},
		[]string{"schedule_name"},
	)

	promScheduleLagSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "schedule_lag_seconds",
			Help:      "Seconds the schedule is overdue (now - tm_next_run) at scan time. Aggregate with max by (schedule_name) across replicas.",
		},
		[]string{"schedule_name"},
	)

	promExecutionRows = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "execution_rows",
			Help:      "Total schedule_executions row count, refreshed once per tick (alarms if execution-retention stops working).",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promDispatchTotal,
		promDispatchDurationMS,
		promScheduleLastSuccessTimestamp,
		promScheduleLagSeconds,
		promExecutionRows,
	)
}
