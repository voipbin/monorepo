// Package monitoringbackend holds the contract every sentinel-manager monitoring backend
// satisfies, plus the metrics that are meaningful regardless of which backend is running
// (design §8.3, §8.4 item 4).
//
// sentinel-manager watches container lifecycles on two very different runtimes: Docker on
// bm-nyc-01 (pkg/dockerwatchhandler) and Kubernetes on a self-hosted cluster
// (pkg/k8swatchhandler). VoIPBin is a self-hostable opensource CPaaS, so a self-hoster running
// Kubernetes needs stranded-call detection exactly as much as bm-nyc-01 does; both backends are
// peers, neither is a fallback for the other.
//
// Exactly ONE backend runs per process, selected once at startup by SENTINEL_BACKEND. That is why
// the shared counters below carry no `backend` label: a label that can only ever hold one value
// for a process's entire lifetime adds cardinality for zero discriminating power (design §8.4
// item 4).
//
// This package must stay free of any backend-specific import (no docker, no k8s.io/*) — both
// backends import it, so an import in the other direction would be a cycle, and a k8s.io/* import
// here would leak that dependency into every module that reaches sentinel-manager through a
// `replace` directive.
package monitoringbackend

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// MonitoringBackend is the one-method contract a monitoring backend satisfies (design §8.3).
//
// Contract, stated explicitly because it is NOT the obvious default and the pre-VOIP-1418 K8s
// implementation violated it: Run returns nil ONLY when ctx was cancelled (normal shutdown). Any
// other cause of the watch loop stopping — an informer sync failure, a watch that dies and cannot
// be re-established, a client losing its connection to the API server, an event stream that stays
// dead — MUST return a non-nil error, which cmd/sentinel-manager propagates to a non-zero process
// exit.
//
// A backend that "looks up" but watches nothing is worse than one that is visibly down: the
// orchestrator reports success, alerting stays quiet, and stranded calls simply never get
// recovered.
type MonitoringBackend interface {
	Run(ctx context.Context) error
}

// list of the container state labels shared by both backends' state-change counter.
const (
	StateStarted = "started"
	StateDied    = "died"
)

// MetricsNamespace is the Prometheus namespace prefix for every sentinel-manager metric.
//
// It lives here rather than in either backend package so both register under an identical prefix.
// When each backend computed its own, nothing structurally prevented the two from silently
// diverging, which would have split one logical counter into two differently-named series.
var MetricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameSentinelManager)

var (
	// PromContainerStateChangeCounter counts published container lifecycle events.
	//
	// Shared, and EXPORTED so both backends can increment it: the labels describe the published
	// event, not the runtime that produced it, so a Docker deployment and a Kubernetes deployment
	// populate the same series and the same Grafana panels. Leaving this Docker-only would have
	// left a Kubernetes deployment's primary dashboard row silently blank — no error, just empty
	// panels, discovered during an incident.
	PromContainerStateChangeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "container_state_change_total",
			Help:      "Counts the number of watched container state changes",
		},
		[]string{"container_name", "service", "state"},
	)

	// PromContainerUnresolvedAsteriskIDCounter counts DEATHS published with an unresolved
	// asterisk-id (design §8.4 item 4: identical metric name across backends, no `backend` label).
	//
	// Downstream, such an event is dropped by bin-call-manager's empty-id guard, so this counter
	// is the only signal that a recovery did not happen. Both backends increment it for the same
	// reason through different mechanisms: the Docker backend when its state table never resolved
	// an id before the container died, the Kubernetes backend when the pod carried no
	// `asterisk-id` annotation at death.
	//
	// Scope is deliberately DEATHS ONLY, not "any event with an empty id". A `container_started`
	// legitimately carries an empty id on both backends — always on the Docker side (a freshly
	// started container has not been resolved yet) and during the annotation-patch window on the
	// Kubernetes side (design §8.2) — and counting those would swamp the signal this metric
	// exists for and make the shipped Grafana panel fire constantly on a healthy cluster.
	PromContainerUnresolvedAsteriskIDCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "container_unresolved_asterisk_id_total",
			Help:      "Counts container_died events published without a resolved asterisk-id",
		},
		[]string{"container_name"},
	)
)

func init() {
	prometheus.MustRegister(
		PromContainerStateChangeCounter,
		PromContainerUnresolvedAsteriskIDCounter,
	)
}
