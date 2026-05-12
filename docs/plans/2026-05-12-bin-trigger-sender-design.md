# Design: bin-trigger-sender

**Date:** 2026-05-12
**Branch:** NOJIRA-Add-bin-trigger-sender

## Problem

The `number-renew` CronJob pulls its container image from `registry.gitlab.com/voipbin/bin-manager/request-sender:latest` and requires a `gitlab-auth` imagePullSecret in Kubernetes. This is the only remaining dependency on GitLab's container registry — every other `bin-*` service image lives on Docker Hub and is built via the existing CircleCI pipeline.

## Goal

Migrate `request-sender` into the monorepo as `bin-trigger-sender`, build and push its image to Docker Hub through the existing CircleCI pipeline, and update the `number-renew` CronJob to use the new image — eliminating the GitLab registry dependency entirely.

## What bin-trigger-sender Does

A minimal CLI tool that sends an RPC-style request message to a RabbitMQ queue. It is designed to run as a Kubernetes CronJob container.

**Flags:**
| Flag | Description |
|------|-------------|
| `-rabbit_addr` | RabbitMQ address (e.g. `amqp://...`) |
| `-queue` | Target queue name |
| `-uri` | Request URI (e.g. `/v1/numbers/renew`) |
| `-method` | HTTP-style method (`POST`, `GET`, etc.) |
| `-data_type` | Content type (e.g. `application/json`) |
| `-data` | Request body as JSON string |
| `-timeout` | Timeout in milliseconds |
| `-delay` | Initial delay before sending, in milliseconds |

No secrets are baked into the image. The RabbitMQ address arrives at runtime via a Kubernetes secret.

## Monorepo Subproject Layout

```
bin-trigger-sender/
├── cmd/
│   └── bin-trigger-sender/
│       └── main.go          # CLI entrypoint
├── Dockerfile
├── go.mod
└── go.sum
```

Follows the same conventions as other `bin-*` services. No `k8s/` subdirectory — deployment is owned by `bin-number-manager/k8s/cronjob.yml`.

## CI Pipeline Changes (CircleCI)

Add three jobs to `.circleci/config_work.yml` mirroring the pattern of every other `bin-*` service:

- **`bin-trigger-sender-test`** — `go test ./...`
- **`bin-trigger-sender-build`** — `docker-build` orb: builds image, pushes `voipbin/bin-trigger-sender:<sha>` (and `:latest` on `main`) to Docker Hub using existing `$CC_DOCKERHUB_USERNAME`/`$CC_DOCKERHUB_PASSWORD` credentials
- **`bin-trigger-sender-release`** — `docker-release` orb: deploys to GKE

Wire into the existing path-filter workflow so jobs only run when `bin-trigger-sender/.*` files change.

## CronJob Changes (`bin-number-manager/k8s/cronjob.yml`)

```yaml
# Before
imagePullSecrets:
- name: gitlab-auth
containers:
- name: request-sender
  image: registry.gitlab.com/voipbin/bin-manager/request-sender:latest

# After
# (imagePullSecrets removed — Docker Hub public image needs no auth)
containers:
- name: bin-trigger-sender
  image: voipbin/bin-trigger-sender:<sha>
```

Pin to the commit SHA tag, not `:latest`, to prevent unintended image drift.

## Security Posture

| Risk | Mitigation |
|------|-----------|
| Public image (binary visible) | Acceptable — no secrets in image; it is a generic AMQP client |
| Image drift / supply chain | Pin CronJob to SHA tag, not `:latest` |
| Docker Hub credential compromise | Existing `$CC_DOCKERHUB_USERNAME`/`$CC_DOCKERHUB_PASSWORD` already scoped to CI; no new credentials needed |

## Migration Steps (high-level)

1. Create `bin-trigger-sender/` subproject — port or rewrite the `request-sender` Go source
2. Add CircleCI jobs and path-filter entries
3. Merge to `main`, verify image appears on Docker Hub
4. Update `bin-number-manager/k8s/cronjob.yml` with new image + SHA tag
5. Deploy and verify the `number-renew` CronJob pulls successfully
6. Retire the GitLab `bin-manager/request-sender` pipeline

## Out of Scope

- Changes to any other CronJob or service
- Modifying the RabbitMQ message protocol
- Adding new flags or behaviour to `bin-trigger-sender`
