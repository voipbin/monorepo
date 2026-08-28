# bin-*-manager Two-Replica Rollout — Fleet Design

Fleet-wide plan for scaling every `bin-*-manager` Komodo stack from 1 to
2 replicas. The durable, service-agnostic recipe (eligibility checklist,
compose change, deploy verification) lives in
[docs/workflows/manager-replica-scaling.md](../workflows/manager-replica-scaling.md);
this document holds the plan-shaped content that expires as the rollout
progresses: gating conditions and the per-service tracker.

Same shape as the Komodo rollout plans that preceded it:
[2026-08-18-bin-manager-komodo-rollout-tier1-design.md](2026-08-18-bin-manager-komodo-rollout-tier1-design.md),
[2026-08-18-bin-manager-komodo-rollout-tier2-design.md](2026-08-18-bin-manager-komodo-rollout-tier2-design.md).

## Context

Every `bin-*-manager` runs as a single container on bm-nyc-01. The goal
is 2 replicas per service, for throughput/load-sharing and faster
recovery after a container restart.

A zero-downtime A/B-pair approach (keep `container_name`, deploy the
pair in stages) was designed and reviewed first, then **deliberately
dropped by CEO/CTO decision in favor of the simple approach**: remove
`container_name`, add `deploy.replicas: 2`, and accept the brief
restart window a redeploy causes. The A/B machinery bought little for
its complexity once downtime was acceptable.

The pilot (P7) is done: `bin-schedule-manager` was converted in
[PR #1212](https://github.com/voipbin/monorepo/pull/1212) and deployed
live on 2026-08-27 — both replicas
(`bin-schedule-manager-schedule-manager-1`/`-2`) came up healthy, the
Komodo/Compose naming behaved as expected, and Prometheus DNS SD picked
up both replicas with no scrape-config change. schedule-manager was
chosen because its dispatch loop is replica-safe by design (Redis lock
+ DB CAS claim) and it uses no per-pod queue.

## Gating conditions

These gate the **actual replica-scaling PRs** for the remaining
services. This design document itself, and the doc-only companion edits
that landed with it, are not gated (no code, no deploy).

1. **BLOCKING — app-side connection-pool caps must be DEPLOYED (not
   merely merged) before any fleet-wide scale-out.**
   `bin-common-handler/pkg/databasehandler`'s `Connect()` sets no pool
   limit (`SetMaxOpenConns` is never called), and `bin-rag-manager`
   opens its own PostgreSQL pool (`sql.Open("postgres", ...)` in
   `cmd/rag-manager/main.go`) with the same unbounded default — so
   every service's connection count is bounded only by load. Doubling
   replicas doubles the theoretical draw against the shared MariaDB.
   `max_connections=220` (raised from the 151 default in monorepo-etc,
   2026-08-27) is headroom, not a fix. Until the pool-cap change is
   verified deployed — check the running images' commit, not the merge
   — scale at most a handful of low-traffic services at a time, never
   the full fleet.
2. **Advisory — a 1-of-2 degradation alert does not exist yet.** With
   replicas, `InstanceDown` (`up == 0`) cannot fire for a lost replica
   (its DNS-SD target simply disappears) and `ManagerServiceGone` fires
   only when *zero* replicas survive. A service silently running 1-of-2
   is invisible today. A `count by (service)
   (up{job="voipbin-managers"}) < 2`-style rule in monorepo-etc's
   alert-rules.yml closes this; land it as the rollout proceeds. Until
   then, the post-deploy manual verification in the workflows doc is
   the only 2-of-2 check.

## Per-service tracker

32 Komodo stacks. States: **done** (scaled and verified live),
**ready** (no blocker; still subject to gating condition 1), **blocked**
(a concrete obstacle must be cleared first), **paced** (safe, but
deliberately scheduled late as a canary).

| Service | State | Blocker / notes |
|---|---|---|
| schedule | **done** | PR #1212, live-verified 2026-08-27 |
| agent, ai, billing, call, campaign, conference, conversation, customer, direct, email, message, number, outdial, queue, registrar, storage, tag, talk, transfer, webchat, webhook | ready | No per-pod queue, no naming dependency, no unguarded global state. Subject to gating condition 1 |
| flow | ready | Has a global mutex map, but it wraps redsync (Redis distributed locks) — replica-safe. Subject to gating condition 1 |
| hook | **blocked** | Caddy naming: monorepo-etc `infra-caddy/config/Caddyfile:77-82` targets the `voipbin-hook-manager` container name literally as its reverse_proxy upstream. Removing `container_name` breaks it. Needs the Caddy retarget companion work first — do NOT fold into the ready batch |
| api | **blocked** (two distinct blockers) | (i) Caddy naming: Caddyfile:17-22 targets `voipbin-api-manager` (retargeted from the old install/-era `voipbin-api-mgr` in VOIP-1362 — cite the current name). (ii) Per-pod `streamData` process-local state needs a code redesign; the Caddy fix alone does not make api scalable. The AudioSocket advertise-address defect is already fixed (`internal/nethandler.AdvertiseIP()`, PR #1209) |
| pipecat | **blocked** | Per-pod queue routing redesign needed; also shares a network namespace 1:1 with its sidecar (`network_mode: service:...`) — the heaviest case in the fleet |
| transcribe | **blocked** | Per-pod queue routing redesign needed (stop-RPCs must reach the owning replica). Note its HostID is a random UUID, not POD_IP |
| tts | **blocked** (lighter than pipecat) | Queue routing is already replica-safe (HOSTNAME-based per-pod queue names separate automatically). Remaining work: the `POD_IP=voipbin-tts-manager-http` sidecar-DNS dependency and the shared-volume/single-http-sidecar layout |
| timeline | **blocked** | In-memory batch buffer can drop events under pressure with no metrics; instrumentation must land first, then re-evaluate |
| route | **blocked** | Global provider healthcheck loop has no cross-replica lock — 2 replicas would double-probe every SIP provider |
| contact | **blocked** | In-process lock needs a Redis distributed-lock (or equivalent) decision first |
| rag | **paced** (not blocked) | Already replica-safe: `DocumentClaimForProcessing` claims work via an atomic CAS UPDATE (same class of safety as schedule's claim machinery). Gating condition 1 applies via rag's own pool-cap fix (it bypasses `databasehandler.Connect()`). Deliberately scheduled as a late canary — this is pacing, not a code defect; do not group it with timeline/route/contact |

## References

- [docs/workflows/manager-replica-scaling.md](../workflows/manager-replica-scaling.md) — the durable recipe
- [PR #1212](https://github.com/voipbin/monorepo/pull/1212) — the pilot (schedule-manager)
- [bin-schedule-manager/docs/operations.md](../../bin-schedule-manager/docs/operations.md) — pilot's service-specific notes (backup-path safety)
- [docs/patterns/per-pod-queues.md](../patterns/per-pod-queues.md) — what makes pipecat/transcribe blocked
- [2026-08-18-bin-manager-komodo-rollout-tier1-design.md](2026-08-18-bin-manager-komodo-rollout-tier1-design.md), [-tier2-](2026-08-18-bin-manager-komodo-rollout-tier2-design.md) — the Komodo migration this builds on
