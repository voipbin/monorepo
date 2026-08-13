# sentinel-manager

# Deploy

Unlike the other `bin-*-manager` services, this one is NOT deployed via
direct SSH to bm-nyc-01 - it requires the Kubernetes API
(`rest.InClusterConfig`) and has no bm-nyc-01 compose-service equivalent
(see `voipbin/voipbin`'s `install/docker-compose.yml.dist` and
`sync-compose-images.sh`'s `LOCK_ONLY_IMAGES` list), so it stays on the
original GKE `bin-sentinel-manager-release` job.

<!-- Updated dependencies: 2026-02-20 -->
