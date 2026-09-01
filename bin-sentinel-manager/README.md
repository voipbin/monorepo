# sentinel-manager

Watches Asterisk container lifecycles and publishes `container_started` /
`container_died` events to RabbitMQ. `bin-call-manager` consumes the death
events and recovers the affected calls.

Two peer backends, selected by `SENTINEL_BACKEND`:

| Value | Watches | Used by |
|---|---|---|
| `docker` | Docker Events API, through a read-only socket proxy | bm-nyc-01 (`komodo/docker-compose.yml`) |
| `kubernetes` | Kubernetes pod informers | self-hosted Kubernetes (self-hoster's own manifests) |

There is no default and no auto-detection: an unset or unknown value fails
startup. Both backends publish the same event schema, so consumers never need
to know which one is running.

# Deploy

**Docker** — deployed like every other `bin-*-manager` service: CircleCI's
`bin-sentinel-manager-deploy` job renders the image tag into
`komodo/docker-compose.yml` and pushes it through the Komodo API.

**Kubernetes** — this repository does not ship or maintain Kubernetes manifests.
A self-hoster running the Kubernetes backend writes their own `Deployment` and
a `pod-reader` `Role`/`RoleBinding` on the `voip` namespace for their own
cluster, and sets `SENTINEL_BACKEND=kubernetes` on it. Without that RBAC the
informer's initial list is denied and the process exits at startup.

VOIP-1418 first replaced the original Kubernetes-only implementation with the
Docker backend (GKE was dismantled on 2026-08-20, leaving the service with no
deploy target), then restored Kubernetes alongside it as a peer — a self-hosted
Kubernetes deployment needs stranded-call detection just as much. See
`docs/plans/2026-09-01-voip-1418-sentinel-docker-backend-design.md`, §8 for the
two-backend addendum. A follow-up removed the example `k8s/*.yml` manifests
this repo had briefly carried: deployment configuration for a self-hoster's own
cluster is theirs to own, not something VoIPBin maintains a reference copy of.

<!-- Updated dependencies: 2026-09-01 -->
