package aicallhandler

import "github.com/prometheus/client_golang/prometheus"

var (
	promAIcallIdleExpiredTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_idle_expired_total",
			Help:      "Total AIcalls terminated due to conversation idle-timeout on reuse path.",
		},
	)
	promAIcallInterruptAttemptedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_interrupt_attempted_total",
			Help:      "Pipecat interrupt attempts on AIcall reuse, by outcome (gone|dead|alive|error).",
		},
		[]string{"result"},
	)
	promAIcallContactCaseRecreateRateLimitedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_contact_case_recreate_rate_limited_total",
			Help:      "Total times a contact_case AIcall recreation was blocked by the post-terminate rate limit (VOIP-1234).",
		},
	)
)

func init() {
	prometheus.MustRegister(promAIcallIdleExpiredTotal, promAIcallInterruptAttemptedTotal, promAIcallContactCaseRecreateRateLimitedTotal)
}

// Test parallelism note:
// promAIcallIdleExpiredTotal and promAIcallInterruptAttemptedTotal are
// package-level Prometheus counters registered in init(). Tests that assert
// counter increments via testutil.ToFloat64 use a before/after snapshot
// pattern. These tests MUST NOT call t.Parallel() at the test function level
// — concurrent increments would corrupt the delta assertion. Sub-tests within
// a single test function are sequential by default and are safe.
//
// Test_aicallHandler_interruptPreviousPipecatcall does not use t.Parallel,
// so all sub-tests run sequentially — snapshots over the {gone|dead|alive|error}
// labels are safe.
//
// Test_startReferenceTypeConversation does call t.Parallel() in sub-tests, but
// the idle-expired counter is incremented by exactly one sub-case (idle-expired
// AIcall — terminate then recreate). Since no other sub-case touches
// promAIcallIdleExpiredTotal, the snapshot pattern remains safe for that one
// sub-test.
