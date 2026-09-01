# sentinel-manager

Watches the Docker Engine's container lifecycle events for the Asterisk
containers on bm-nyc-01 and publishes `container_started` / `container_died`
events to RabbitMQ. `bin-call-manager` consumes the death events and recovers
the affected calls.

# Deploy

Deployed like every other `bin-*-manager` service: CircleCI's
`bin-sentinel-manager-deploy` job renders the image tag into
`komodo/docker-compose.yml` and pushes it through the Komodo API.

Until VOIP-1418 this service was Kubernetes-only (`rest.InClusterConfig`) and
had no deploy target after GKE was dismantled on 2026-08-20. It now runs on
Docker, talking to a read-only `docker-socket-proxy` sidecar rather than the
raw socket. See
`docs/plans/2026-09-01-voip-1418-sentinel-docker-backend-design.md`.

<!-- Updated dependencies: 2026-09-01 -->
