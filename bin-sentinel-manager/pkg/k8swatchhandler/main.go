// Package k8swatchhandler watches Kubernetes pod lifecycles and republishes the transitions that
// matter for call recovery (design §8).
//
// It is the peer of pkg/dockerwatchhandler, not a fallback for it: VoIPBin is a self-hostable
// opensource CPaaS, and a self-hoster running Kubernetes needs stranded-call detection exactly as
// much as the Docker deployment on bm-nyc-01 does. Exactly one backend runs per process, chosen at
// startup by SENTINEL_BACKEND.
//
// # Why this is much simpler than the Docker backend
//
// The Docker backend's hardest problem — resolving a dying container's asterisk-id, when that id
// derives from a MAC address that is not stable across recreation — does not exist here.
// voip-asterisk-proxy self-patches its own pod's `asterisk-id` annotation through the Kubernetes
// API at startup, and a pod's annotations are visible to any watcher with read RBAC. So there is
// no reverse lookup, no state table, no freshness filter, and no sticky-last-known semantics: the
// id is simply read off the pod object (design §8.2).
//
// # What is NOT simple here
//
// Three things in this package are load-bearing and were each caught by a distinct design-review
// round as missing from a naive "restore the old code" approach. All three fail SILENTLY if they
// regress — no panic, no error, just a death event that never gets published:
//
//  1. UpdateFunc's UID-mismatch check (run.go) — a same-name pod replaced during a watch
//     interruption never produces a delete callback at all.
//  2. DeleteFunc's DeletedFinalStateUnknown unwrap (run.go) — a deletion discovered on relist
//     arrives wrapped in a tombstone, and a bare type assertion panics on it.
//  3. Fail-loud propagation (run.go) — the pre-VOIP-1418 code spawned informers in bare
//     goroutines and returned nil unconditionally, which is precisely the "looks up, watches
//     nothing, exits 0" mode this service must never be in.
package k8swatchhandler

//go:generate mockgen -package k8swatchhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/monitoringbackend"
)

// pod metadata keys this backend reads.
const (
	// podLabelApp identifies the Asterisk workload. Its value is mapped through
	// serviceByPodLabelApp, never passed through unmapped.
	podLabelApp = "app"

	// podAnnotationAsteriskID is the annotation voip-asterisk-proxy self-patches at startup. A
	// pod that dies before that patch lands genuinely has no id -- an expected degrade path, not
	// a corner case (design §8.2).
	podAnnotationAsteriskID = "asterisk-id"
)

// watchedNamespace is the namespace the Asterisk pods run in. Kept a compile-time constant, the
// same way dockerwatchhandler keeps its watched container-name prefixes: the design deliberately
// chose not to make the watch set runtime config on either backend.
const watchedNamespace = "voip"

// serviceByPodLabelApp maps a pod's `app` label to the typed service constant published on
// container.Event.
//
// This lookup is REQUIRED to be explicit, not a passthrough (design §8.3). Event.Service is a
// typed constant on the Docker side, and bin-call-manager filters on it exactly; assigning a raw
// label value means a typo'd or unexpected `app` label silently produces an event whose filter
// never matches, with nothing anywhere reporting that something went wrong. Anything not in this
// map is rejected at the publish boundary -- log and skip -- mirroring how the Docker backend
// ignores container names that do not match its watched prefixes.
var serviceByPodLabelApp = map[string]string{
	"asterisk-call":       smcontainer.ServiceAsteriskCall,
	"asterisk-conference": smcontainer.ServiceAsteriskConference,
	"asterisk-registrar":  smcontainer.ServiceAsteriskRegistrar,
}

// watchTarget is one (namespace, label-selector) pair, each of which gets its own informer.
type watchTarget struct {
	Namespace     string
	LabelSelector string
}

// watchTargets are the exact selectors the pre-VOIP-1418 cmd/sentinel-manager hardcoded. Pinned by
// Test_watchTargets_matchTheServiceMap so this list and serviceByPodLabelApp cannot drift apart.
var watchTargets = []watchTarget{
	{Namespace: watchedNamespace, LabelSelector: "app=asterisk-call"},
	{Namespace: watchedNamespace, LabelSelector: "app=asterisk-conference"},
	{Namespace: watchedNamespace, LabelSelector: "app=asterisk-registrar"},
}

const (
	// cacheSyncTimeout bounds the initial informer sync.
	//
	// A bare WaitForCacheSync is NOT enough, and the reason is subtle: it returns false only when
	// its stop channel closes. Under the canonical failure this backend must fail loud on --
	// missing `pod-reader` RBAC -- the reflector's initial List is denied and client-go retries
	// with backoff FOREVER, so the call simply blocks and never returns false on its own. That is
	// how the old code could claim "RBAC required or the service exits" while actually hanging.
	// The deadline is what makes that claim true (design §8.4 item 3).
	cacheSyncTimeout = 60 * time.Second

	// maxConsecutiveWatchFailures is the fail-loud budget for watch errors, the K8s analogue of
	// dockerwatchhandler's maxConsecutiveEmptyStreams.
	//
	// SetWatchErrorHandler fires on entirely benign conditions too -- an apiserver rolling
	// restart, a `too old resource version` forcing a relist, a transient connection reset --
	// so treating any single invocation as fatal would self-restart a healthy system, while
	// treating them all as "just log" reproduces exactly the silent-failure behavior this
	// rewrite exists to remove. Only a sustained run of failures with no intervening recovery
	// converts to the fatal error Run returns.
	maxConsecutiveWatchFailures = 20

	// watchHealthInterval is how often Run re-checks the reflector's last synced resource
	// version.
	//
	// Delivered events are the primary "the watch is healthy" signal, but they are not
	// sufficient: a selector matching zero pods delivers nothing even when the watch is perfectly
	// healthy, and that must not drain the budget. A resource-version change proves the reflector
	// completed a list/watch regardless of whether any object matched.
	watchHealthInterval = 10 * time.Second
)

// list of the watch-health outcome labels (design §8.4 item 3's three-valued outcome).
const (
	// watchOutcomeResynced marks a recovery: the reflector completed a list/watch, or delivered an
	// event, after at least one failure. Resets the budget.
	watchOutcomeResynced = "resynced"
	// watchOutcomeTransientError marks a single watch error that has not exhausted the budget.
	watchOutcomeTransientError = "transient-error"
	// watchOutcomeFatal marks budget exhaustion, which is what Run converts into a returned error.
	watchOutcomeFatal = "fatal"
)

// list of the died-detection source labels (design §8.4 items 2-3: tombstone-recovered and
// UID-mismatch deaths go on one shared delete-path counter, since both are equally a signal of
// watch instability).
const (
	// diedSourceLive is the normal path: a delete callback carrying a real pod object.
	diedSourceLive = "live"
	// diedSourceTombstone is a deletion missed during a watch interruption and only discovered on
	// the next relist, delivered as cache.DeletedFinalStateUnknown.
	diedSourceTombstone = "tombstone"
	// diedSourceUIDMismatch is a death inferred from a same-key pod replacement seen in
	// UpdateFunc, for which no delete callback ever fires.
	diedSourceUIDMismatch = "uid-mismatch"
	// diedSourceUnrecoverable is a delete callback whose payload could not be resolved to a pod at
	// all. Nothing can be published, so this counter is the ONLY trace that a death was observed
	// and lost -- it must never be a silent return.
	diedSourceUnrecoverable = "unrecoverable"
)

type k8sWatchHandler struct {
	util          utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	clientset kubernetes.Interface

	cacheSyncTimeout    time.Duration
	watchHealthInterval time.Duration
	maxWatchFailures    int
}

// K8sWatchHandler watches pod lifecycle transitions and publishes them.
type K8sWatchHandler interface {
	Run(ctx context.Context) error
}

// this backend must satisfy the shared contract cmd/sentinel-manager selects on.
var _ monitoringbackend.MonitoringBackend = (K8sWatchHandler)(nil)

// NewK8sWatchHandler creates a K8sWatchHandler.
//
// The clientset is INJECTED rather than built here from rest.InClusterConfig(): the composition
// root (cmd/sentinel-manager) constructs the real one, which keeps this package testable against
// client-go's fake clientset. That mirrors how the Docker backend already receives its Docker and
// Redis clients.
func NewK8sWatchHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler utilhandler.UtilHandler,
	clientset kubernetes.Interface,
) K8sWatchHandler {
	h := &k8sWatchHandler{
		util:          utilHandler,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		clientset: clientset,

		cacheSyncTimeout:    cacheSyncTimeout,
		watchHealthInterval: watchHealthInterval,
		maxWatchFailures:    maxConsecutiveWatchFailures,
	}

	return h
}

// mapService resolves a pod's `app` label to the typed service constant, reporting false for any
// value this backend does not watch.
func mapService(labelApp string) (string, bool) {
	service, ok := serviceByPodLabelApp[labelApp]
	return service, ok
}

var (
	// promWatchHealthCounter makes watch instability observable rather than log-only.
	//
	// The Docker backend has container_event_stream_reconnect_total for the same purpose; this is
	// its K8s counterpart. A rising `transient-error` rate with matching `resynced` events is a
	// flapping-but-surviving watch that may be losing deltas in the gaps; `fatal` means the budget
	// was exhausted and the process is exiting.
	promWatchHealthCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: monitoringbackend.MetricsNamespace,
			Name:      "pod_watch_health_total",
			Help:      "Counts pod watch health transitions by outcome",
		},
		[]string{"namespace", "selector", "outcome"},
	)

	// promDiedDetectionCounter records HOW each death was detected.
	//
	// `live` is the healthy path. A spike in `tombstone` or `uid-mismatch` means deaths are being
	// recovered from relists rather than seen on the live watch -- the events still publish, so
	// nothing is lost, but it is direct evidence of watch instability and worth alerting on before
	// something IS lost. `unrecoverable` means a death was observed and could NOT be published.
	promDiedDetectionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: monitoringbackend.MetricsNamespace,
			Name:      "pod_died_detection_total",
			Help:      "Counts published container_died events by how the death was detected",
		},
		[]string{"source"},
	)
)

func init() {
	prometheus.MustRegister(
		promWatchHealthCounter,
		promDiedDetectionCounter,
	)
}
