package aicallhandler

// The aicall_backstop_reply_total counter previously declared here was relocated
// to pkg/messagehandler (see metrics_backstop.go) because the only emit site is
// messagehandler.EventPMPipecatcallTerminated, and aicallhandler imports
// messagehandler — referencing the counter here would have created a circular
// import. The fully-qualified metric name (`ai_manager_aicall_backstop_reply_total`)
// is preserved by reusing the same `metricsNamespace` from this package's main.go.
