// Package dockerwatchhandler watches the Docker Engine's container lifecycle events and
// republishes the ones that matter for call recovery (VOIP-1418).
//
// It replaces the former pkg/monitoringhandler, which drove a Kubernetes SharedIndexInformer.
// The rename is deliberate: "monitoring" was K8s-flavoured terminology for what is really a
// Docker event watcher, and this is a full rewrite rather than a patch.
//
// The non-obvious part of this package is the state table (state.go / refresh.go): a container's
// asterisk-id CANNOT be resolved at the moment it dies -- a dead container's inspect response has
// an empty IPAddress, and a reverse Redis scan at die time cannot tell "the id that just died"
// from "the id that just took over the same static IP". The id is therefore resolved
// continuously, BEFORE the death, and merely read at die time.
package dockerwatchhandler

//go:generate mockgen -package dockerwatchhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sort"
	"strings"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerevents "github.com/docker/docker/api/types/events"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/prometheus/client_golang/prometheus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/cachehandler"
)

// watchedContainerPrefixes maps a watched container-name prefix to the logical service behind it.
//
// These stay compile-time constants, exactly like the K8s label selectors they replace: the
// design explicitly chose not to promote them to runtime config (design §3.1). The trailing dash
// is part of the prefix -- see matchWatchedContainer for why the remainder must then be a bare
// replica index.
var watchedContainerPrefixes = map[string]string{
	"voip-asterisk-call-docker-":       smcontainer.ServiceAsteriskCall,
	"voip-asterisk-conference-docker-": smcontainer.ServiceAsteriskConference,
	"voip-asterisk-registrar-docker-":  smcontainer.ServiceAsteriskRegistrar,
}

const (
	// refreshInterval is how often the background loop re-reads Redis to resolve table entries'
	// asterisk-ids. It is INDEPENDENT of the event stream and unrelated to the proxy's own 5-min
	// key-refresh cadence (which is what the freshness filter is keyed to instead).
	refreshInterval = 10 * time.Second

	// reconnectDelay is the pause before re-opening a dropped Docker event stream. The stream is
	// resumed with `since=<last processed event>`, so the gap is bounded rather than lossy.
	reconnectDelay = 3 * time.Second

	// maxConsecutiveEmptyStreams bounds how long sentinel will keep silently retrying a dead
	// event stream before giving up and exiting non-zero.
	//
	// A HEALTHY stream never returns: the Docker Events API blocks until the context is cancelled
	// or the connection breaks, so an idle fleet does not tick this counter. An attempt that ends
	// WITHOUT having delivered a single message therefore means the stream could not be
	// established (or died instantly) -- the socket proxy is gone, the ACL changed, the network
	// is partitioned. Any attempt that delivers at least one event resets the counter to zero.
	//
	// 20 attempts at reconnectDelay = roughly a minute of continuous failure, which comfortably
	// outlasts a proxy restart or a compose redeploy while still failing fast enough that Komodo
	// shows a crash-loop rather than a container sitting "up" and watching nothing. Retrying
	// forever with only a log line is exactly the failure mode design §3.2 calls worse than being
	// visibly down.
	maxConsecutiveEmptyStreams = 20

	// flapWindow / flapThreshold damp a crash-looping container: past flapThreshold deaths inside
	// flapWindow, further deaths in that window are logged but NOT published. Repeatedly firing
	// recovery against a container stuck in a crash-loop just spams Homer/PJSIP for channels that
	// likely never established. A flapping container is a symptom to alert on, not something to
	// keep redialing against.
	flapWindow    = 60 * time.Second
	flapThreshold = 3

	// dockerNetworkProduction is the shared Docker network the watched containers hold their
	// static IPs on. It is only a PREFERENCE for IP resolution, not a requirement -- see
	// resolveContainerIP.
	dockerNetworkProduction = "production"
)

// dockerClient is the narrow subset of the Docker Engine API this handler uses.
//
// It is deliberately three methods wide. Everything here is a GET; the docker-socket-proxy in
// front of the real socket is configured to allow exactly the EVENTS and CONTAINERS API families
// and deny every mutating one, so a method added here that the proxy does not allow would fail at
// runtime rather than compile time. Keep the interface and the proxy ACL in step.
type dockerClient interface {
	ContainerList(ctx context.Context, options dockercontainer.ListOptions) ([]dockercontainer.Summary, error)
	ContainerInspect(ctx context.Context, containerID string) (dockercontainer.InspectResponse, error)
	Events(ctx context.Context, options dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error)
}

type dockerWatchHandler struct {
	util          utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	dockerClient dockerClient
	cacheHandler cachehandler.CacheHandler

	state *stateTable
	flap  *flapTracker

	refreshInterval time.Duration
	reconnectDelay  time.Duration
}

// DockerWatchHandler watches container lifecycle transitions and publishes them.
type DockerWatchHandler interface {
	Run(ctx context.Context) error
}

// NewDockerWatchHandler creates a DockerWatchHandler.
func NewDockerWatchHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler utilhandler.UtilHandler,
	dockerClient dockerClient,
	cacheHandler cachehandler.CacheHandler,
) DockerWatchHandler {
	h := &dockerWatchHandler{
		util:          utilHandler,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		dockerClient: dockerClient,
		cacheHandler: cacheHandler,

		state: newStateTable(),
		flap:  newFlapTracker(flapWindow, flapThreshold),

		refreshInterval: refreshInterval,
		reconnectDelay:  reconnectDelay,
	}

	return h
}

// matchWatchedContainer resolves a container name to the service it belongs to.
//
// A bare prefix match is NOT enough: the asterisk-proxy sidecars are named after their parent
// (`voip-asterisk-call-docker-1-asterisk-call-proxy-1`) and therefore share the prefix. Recovery
// only cares about the main asterisk container's lifecycle -- matching today's pod-level
// granularity -- so the remainder after the prefix must be a bare Compose replica index.
func matchWatchedContainer(name string) (string, bool) {
	name = strings.TrimPrefix(name, "/")

	for prefix, service := range watchedContainerPrefixes {
		remainder, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}

		if !isReplicaIndex(remainder) {
			// a sidecar or any other derived container sharing the prefix.
			continue
		}

		return service, true
	}

	return "", false
}

// isReplicaIndex reports whether s is a non-empty run of digits.
func isReplicaIndex(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// containerNameOf extracts the canonical container name from a Docker container summary. Docker
// reports names with a leading "/" and a container may carry several; the first is its own.
func containerNameOf(summary dockercontainer.Summary) string {
	if len(summary.Names) == 0 {
		return ""
	}

	return strings.TrimPrefix(summary.Names[0], "/")
}

// resolveContainerIP picks the container's internal IP out of an inspect response.
//
// The `production` network is preferred because that is where the watched containers hold their
// static IPs, but any attached network's address is accepted as a fallback rather than failing
// outright -- a network rename must degrade to "resolved from another network", not to "no IP at
// all, container permanently unresolvable". An empty result is returned when the container has no
// address anywhere, which is the normal state of a DEAD container and the exact reason inspect is
// never called at die time.
func resolveContainerIP(inspect dockercontainer.InspectResponse) string {
	if inspect.NetworkSettings == nil {
		return ""
	}

	if endpoint, ok := inspect.NetworkSettings.Networks[dockerNetworkProduction]; ok && endpoint != nil && endpoint.IPAddress != "" {
		return endpoint.IPAddress
	}

	// deterministic fallback over the remaining networks. The legacy top-level
	// NetworkSettings.IPAddress is deliberately NOT consulted -- it is deprecated for removal in
	// Docker v29 and only ever mirrors the default bridge, which is already present in Networks.
	for _, name := range sortedNetworkNames(inspect.NetworkSettings.Networks) {
		endpoint := inspect.NetworkSettings.Networks[name]
		if endpoint != nil && endpoint.IPAddress != "" {
			return endpoint.IPAddress
		}
	}

	return ""
}

// sortedNetworkNames returns the network names in a stable order, so that a container attached to
// several networks always resolves to the SAME address across passes. Map iteration order would
// make the resolved IP flap between networks, which the state table would then read as a genuine
// change.
func sortedNetworkNames(networks map[string]*dockernetwork.EndpointSettings) []string {
	names := make([]string, 0, len(networks))
	for name := range networks {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameSentinelManager)

	// promContainerStateChangeCounter replaces the former
	// `sentinel_manager_pod_state_change_total`. This is a rename, not an addition: "pod" is
	// equally misleading here, and the `namespace`/`pod` labels it carried have no Docker
	// equivalent. monitoring/grafana/dashboards/sentinel-manager.json is updated in the same
	// change -- a dashboard left on the old name goes silently blank rather than erroring.
	promContainerStateChangeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "container_state_change_total",
			Help:      "Counts the number of watched container state changes",
		},
		[]string{"container_name", "service", "state"},
	)

	// promContainerUnresolvedAsteriskIDCounter counts deaths published with an UNRESOLVED
	// asterisk-id. Downstream, such an event is dropped by call-manager's empty-id guard, so this
	// counter is the only signal that a recovery did not happen.
	promContainerUnresolvedAsteriskIDCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "container_unresolved_asterisk_id_total",
			Help:      "Counts container_died events published without a resolved asterisk-id",
		},
		[]string{"container_name"},
	)

	// promContainerRefreshMissCounter counts refresh passes that found NO fresh candidate for a
	// container whose asterisk-id was already resolved. The id is deliberately kept (sticky
	// last-known), so this is not itself a failure -- it is the LEADING indicator that the next
	// death for this container may go unrecovered, which is worth alerting on before it happens
	// rather than after.
	promContainerRefreshMissCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "container_asterisk_id_refresh_miss_total",
			Help:      "Counts refresh passes that found no fresh asterisk address for an already-resolved container",
		},
		[]string{"container_name"},
	)

	// promContainerEventStreamReconnectCounter makes post-boot stream loss observable.
	//
	// The boot-time failure path is loud by construction (the process exits), but a proxy that
	// dies AFTER boot would otherwise only produce log lines, leaving sentinel up and watching
	// nothing with no metric to alert on. `result="empty"` counts an attempt that ended without
	// delivering a single event -- the signature of an unreachable stream. A sustained rate of
	// those is the alert; maxConsecutiveEmptyStreams of them in a row exits the process.
	promContainerEventStreamReconnectCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "container_event_stream_reconnect_total",
			Help:      "Counts docker event stream attempts that ended, by whether the attempt delivered any event",
		},
		[]string{"result"},
	)
)

func init() {
	prometheus.MustRegister(
		promContainerStateChangeCounter,
		promContainerUnresolvedAsteriskIDCounter,
		promContainerRefreshMissCounter,
		promContainerEventStreamReconnectCounter,
	)
}
