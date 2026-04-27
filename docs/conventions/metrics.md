# Prometheus Metrics

### 14.1 Service Metrics Registration

Register metrics via `init()` in the `metricshandler` package:

```go
// CORRECT — pkg/metricshandler/main.go
var (
    CallCreateTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "call_manager",
            Name:      "call_create_total",
            Help:      "Total number of call create operations",
        },
        []string{"status"},
    )
)

func init() {
    prometheus.MustRegister(CallCreateTotal)
}
```

### 14.2 Avoid Name Collisions

The shared `requesthandler` auto-registers these metrics per service:
- `<namespace>_request_process_time`
- `<namespace>_event_publish_total`

**NEVER reuse these names** in service-level `metricshandler`:

```go
// WRONG — collides with requesthandler's event_publish_total → PANIC at startup
EventPublishTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "agent_manager",
        Name:      "event_publish_total",  // Already registered!
    },
    []string{"type"},
)

// CORRECT — use unique name
ServiceEventTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "agent_manager",
        Name:      "service_event_total",  // Unique
    },
    []string{"type"},
)
```

**Before adding metrics:** Check `bin-common-handler/pkg/requesthandler/main.go` `initPrometheus()` for existing names.

---
