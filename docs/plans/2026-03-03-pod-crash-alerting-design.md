# Pod Crash Alerting Design

**Date:** 2026-03-03
**Branch:** NOJIRA-add-pod-crash-alerts

## Problem Statement

We have no alerting or metrics for pod crashes across the platform. When a pod crashes, restarts, or goes down, there is no automated notification. Operators must manually check `kubectl` to discover issues.

## Current State

- **Prometheus** deployed in `infrastructure` namespace, scrapes pods via annotations on port 2112
- **AlertManager** deployed with Discord webhook (Slack-compatible API) already configured
- **Grafana** deployed with persistent storage, dashboards in `monitoring/grafana/dashboards/`
- **Existing alerts:** InstanceDown, KamailioDown, RabbitMQ, Redis, CPU/Memory/Disk, RTPEngine
- **No kube-state-metrics** — Kubernetes pod state is not exposed as Prometheus metrics
- **sentinel-manager** watches Asterisk pods in `voip` namespace for application-level recovery (different concern)

## Approach

Deploy **kube-state-metrics** to expose Kubernetes object states as Prometheus metrics. Add Prometheus alert rules for pod crash/down scenarios. Create a Grafana dashboard for visibility.

### Why kube-state-metrics (not extending sentinel-manager)

- kube-state-metrics is the standard, well-maintained Kubernetes component for this purpose
- Zero custom Go code needed
- Covers all namespaces automatically (voip + bin-manager)
- sentinel-manager serves a different purpose (RabbitMQ events for call recovery)
- kube-state-metrics exposes dozens of useful metrics beyond just crashes

## Changes

### 1. Deploy kube-state-metrics

Create Kubernetes manifests in `monorepo-etc/infra-prometheus/k8s_kube_state_metrics/`:
- `deployment.yml` — kube-state-metrics deployment in `infrastructure` namespace
- `service.yml` — ClusterIP service with Prometheus scrape annotations
- `rbac.yml` — ClusterRole + ClusterRoleBinding for reading cluster state
- `kustomization.yml` — Kustomize root (with image override via Kustomize `images`)

Key metrics exposed:
- `kube_pod_container_status_restarts_total` — restart count per container
- `kube_pod_container_status_waiting_reason` — CrashLoopBackOff, ImagePullBackOff
- `kube_pod_container_status_terminated_reason` — OOMKilled, Error
- `kube_deployment_status_replicas_available` — available replicas per deployment
- `kube_deployment_spec_replicas` — desired replicas per deployment

### 2. Prometheus Alert Rules

Add pod-health rule group to `monorepo-etc/infra-prometheus/k8s_prometheus/alert-rules-config-map.yml`. New rules:

| Alert | Expression | For | Severity |
|-------|-----------|-----|----------|
| PodCrashLooping | `increase(kube_pod_container_status_restarts_total{namespace=~"bin-manager\|voip"}[15m]) > 3` | 5m | critical |
| PodOOMKilled | `kube_pod_container_status_terminated_reason{reason="OOMKilled",namespace=~"bin-manager\|voip"} > 0` | 1m | critical |
| PodNotReady | `kube_pod_status_ready{condition="true",namespace=~"bin-manager\|voip"} == 0` | 5m | critical |
| PodDown | `kube_deployment_status_replicas_available{namespace=~"bin-manager\|voip"} == 0` | 2m | critical |

Alerts auto-flow to Discord via the existing AlertManager webhook.

### 3. Grafana Dashboard

Create `monorepo/monitoring/grafana/dashboards/pod-crash-overview.json` with panels:
- Pod restart counts by service (bar chart, last 24h)
- Restart timeline (time series graph)
- OOMKill events table
- Deployment availability (available vs desired replicas)
- Currently crashing pods (stat panel)
- Namespace breakdown (voip vs bin-manager)

### 4. Update Prometheus ConfigMap

The existing `prometheus-server-conf` ConfigMap already has a `kubernetes-service-endpoints` job that scrapes services with `prometheus.io/scrape: "true"` annotation. kube-state-metrics service will include this annotation, so Prometheus will auto-discover and scrape it.

The `prometheus-alert-rules` ConfigMap needs a new rule group for pod health.

## Files Created/Modified

**monorepo-etc** (infra-prometheus):
| File | Action |
|------|--------|
| `infra-prometheus/k8s_kube_state_metrics/deployment.yml` | Create |
| `infra-prometheus/k8s_kube_state_metrics/service.yml` | Create |
| `infra-prometheus/k8s_kube_state_metrics/rbac.yml` | Create |
| `infra-prometheus/k8s_kube_state_metrics/kustomization.yml` | Create |
| `infra-prometheus/k8s_prometheus/alert-rules-config-map.yml` | Modify (add pod-health group) |

**monorepo** (monitoring):
| File | Action |
|------|--------|
| `monitoring/grafana/dashboards/pod-crash-overview.json` | Create |
| `docs/plans/2026-03-03-pod-crash-alerting-design.md` | Create |

## Deployment Steps

1. Apply kube-state-metrics manifests: `kubectl apply -k infra-prometheus/k8s_kube_state_metrics/`
2. Verify kube-state-metrics is running: `kubectl get pods -n infrastructure | grep kube-state`
3. Verify Prometheus scrapes it: check Prometheus targets page
4. Apply updated alert rules: `kubectl apply -k infra-prometheus/k8s_prometheus/`
5. Reload Prometheus config (restart pod or wait for ConfigMap propagation)
6. Import Grafana dashboard JSON from `monitoring/grafana/dashboards/pod-crash-overview.json`
7. Verify alerts fire correctly (optional: test by killing a pod)

## Risks and Mitigations

- **kube-state-metrics RBAC too broad:** Mitigated by using the official minimal ClusterRole
- **Alert noise from expected restarts (e.g., deployments):** Mitigated by `for` duration on alerts
- **Prometheus scrape load:** kube-state-metrics is lightweight; 5s scrape interval is fine
