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

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/cachehandler"
	"monorepo/bin-sentinel-manager/pkg/monitoringbackend"
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
	// The Docker Events API blocks until the context is cancelled or the connection breaks, so a
	// SHORT-LIVED attempt that ends without delivering a single message means the stream could not
	// be established at all -- the socket proxy is gone, the ACL changed, the network is
	// partitioned. Two things reset the counter to zero: an attempt that delivers at least one
	// event, and an attempt that merely SURVIVED a while (healthyStreamLifetimeFactor), which is
	// what an idle fleet's stream looks like when the proxy is restarted underneath it.
	//
	// 20 attempts at reconnectDelay = roughly a minute of continuous failure, which comfortably
	// outlasts a proxy restart or a compose redeploy while still failing fast enough that Komodo
	// shows a crash-loop rather than a container sitting "up" and watching nothing. Retrying
	// forever with only a log line is exactly the failure mode design §3.2 calls worse than being
	// visibly down.
	maxConsecutiveEmptyStreams = 20

	// healthyStreamLifetimeFactor derives, from reconnectDelay, how long a stream must survive
	// before its ending is treated as normal rather than as a failure.
	//
	// Counting only deliveries as "healthy" is not enough: on a genuinely idle fleet no container
	// starts or dies for hours, so a stream can legitimately live a long time and then end
	// (proxy restart, daemon reload) having delivered nothing. Without this, those endings would
	// accumulate on consecutiveEmpty across days and eventually trip the give-up exit with no
	// ongoing fault at all -- a self-inflicted restart on a perfectly healthy system.
	//
	// Connection LONGEVITY is the discriminator. A stream that cannot be established fails almost
	// immediately (connection refused, 403 from the proxy ACL), so a short-lived empty attempt is
	// the real fault signature, while a long-lived one is just an idle watcher being interrupted.
	// 10x reconnectDelay (30s) is comfortably longer than any failure-path round trip and far
	// shorter than a real idle stream's lifetime, so the two cases do not overlap in practice.
	healthyStreamLifetimeFactor = 10

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

	// healthyStreamLifetime is how long an event stream must survive before its ending counts as
	// normal rather than as a failed connection attempt. A zero value DISABLES the longevity
	// reset (every eventless stream then counts toward the give-up budget); the constructor
	// always sets it, so zero only occurs in tests that build the struct directly and do not
	// exercise this path.
	healthyStreamLifetime time.Duration
}

// DockerWatchHandler watches container lifecycle transitions and publishes them.
type DockerWatchHandler interface {
	Run(ctx context.Context) error
}

// this backend must satisfy the shared contract cmd/sentinel-manager selects on.
var _ monitoringbackend.MonitoringBackend = (DockerWatchHandler)(nil)

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

		refreshInterval:       refreshInterval,
		reconnectDelay:        reconnectDelay,
		healthyStreamLifetime: healthyStreamLifetimeFactor * reconnectDelay,
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

// promContainerStateChangeCounter and the unresolved-asterisk-id counter used to live here.
// Both moved to pkg/monitoringbackend once the Kubernetes backend arrived: they describe the
// PUBLISHED EVENT, not the runtime that produced it, so both backends must populate the same
// series or a Kubernetes deployment's dashboards go silently blank. metricsNamespace moved with
// them so neither backend can drift onto a different prefix (design §8.4 item 4).
//
// The counters BELOW stay here on purpose: each describes a mechanism that exists only on the
// Docker side (the Redis-backed asterisk-id state table, and the Docker event stream). The
// Kubernetes backend has no equivalent of any of them and registers its own instead.
var (
	// promContainerRefreshMissCounter counts refresh passes that found NO fresh candidate for a
	// container whose asterisk-id was already resolved. The id is deliberately kept (sticky
	// last-known), so this is not itself a failure -- it is the LEADING indicator that the next
	// death for this container may go unrecovered, which is worth alerting on before it happens
	// rather than after.
	promContainerRefreshMissCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: monitoringbackend.MetricsNamespace,
			Name:      "container_asterisk_id_refresh_miss_total",
			Help:      "Counts refresh passes that found no fresh asterisk address for an already-resolved container",
		},
		[]string{"container_name"},
	)

	// promContainerAsteriskIDConflictCounter counts refresh passes that resolved a DIFFERENT
	// asterisk-id for a container that already had one, and kept the existing id (refresh.go).
	//
	// The fixed-MAC-per-generation invariant says this cannot happen, so any increment is either a
	// real anomaly or a latent bug -- and it has a concrete, plausible trigger: a missed die+start
	// pair (an event-stream gap, with the replacement container reusing the same static IP) leaves
	// the entry holding the DEAD generation's id, and the next death then publishes a wrong
	// asterisk-id. Keeping the existing id is still the right conservative default, but this
	// branch must be alertable, not merely logged.
	promContainerAsteriskIDConflictCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: monitoringbackend.MetricsNamespace,
			Name:      "container_asterisk_id_conflict_total",
			Help:      "Counts refresh passes that resolved a different asterisk-id for an already-resolved container and kept the existing one",
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
			Namespace: monitoringbackend.MetricsNamespace,
			Name:      "container_event_stream_reconnect_total",
			Help:      "Counts docker event stream attempts that ended, by whether the attempt delivered any event",
		},
		[]string{"result"},
	)
)

func init() {
	prometheus.MustRegister(
		promContainerRefreshMissCounter,
		promContainerAsteriskIDConflictCounter,
		promContainerEventStreamReconnectCounter,
	)
}
