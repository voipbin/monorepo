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

1. **RESOLVED by decision (2026-08-28) — app-side pool caps will NOT
   ship; monitoring is the standing defense line.** The original
   condition blocked fleet-wide scale-out until an app-side pool cap
   (`SetMaxOpenConns` in `bin-common-handler/pkg/databasehandler`'s
   `Connect()`, plus `bin-rag-manager`'s own PostgreSQL pool) was
   deployed — pools are unbounded, so doubling replicas doubles the
   theoretical draw on the shared MariaDB. On 2026-08-28 대표님 decided
   to skip the app-side cap entirely. The replacement gate, measured
   the same day: `max_connections=220` with a 7-day peak of 61
   connections (current 39, 24h avg ~37) — a full 2x of the whole
   fleet's peak lands at ~122/220 (55%), and the deployed
   `MySQLConnectionsHigh` (>80% warn, 5m) / `MySQLConnectionsCritical`
   (>90% crit, 2m) → Discord alarm chain catches runaway growth. This
   unblocks the full-fleet P11 rollout; connection headroom is now a
   monitored budget, not a code precondition.
2. **Advisory — a 1-of-2 degradation alert lands WITH the P11 rollout
   window.** With replicas, `InstanceDown` (`up == 0`) cannot fire for
   a lost replica (its DNS-SD target simply disappears) and
   `ManagerServiceGone` fires only when *zero* replicas survive. The
   `ReplicaDegraded` rule (`count by (service) (up{...}) < 2`, warning,
   for 10m, enumerating the replica-scaled services) ships in
   monorepo-etc's alert-rules.yml as P11's companion PR, deployed
   before the compose changes with a pre-created Alertmanager silence
   during the rollout. Until that deploy is verified, the post-deploy
   manual verification in the workflows doc is the only 2-of-2 check.

## Per-service tracker

32 Komodo stacks. States: **done** (scaled and verified live; the
variant "done (P11; deploy verification pending)" means the compose
change is merged but the live 2-of-2 verification has not been recorded
yet), **ready** (no blocker), **blocked** (a concrete obstacle must be
cleared first), **paced** (safe, but deliberately scheduled late as a
canary).

| Service | State | Blocker / notes |
|---|---|---|
| schedule | **done** | PR #1212, live-verified 2026-08-27 |
| agent, ai, billing, call, campaign, conference, conversation, customer, direct, email, message, number, outdial, queue, registrar, storage, tag, talk, transfer, webchat, webhook | **done (P11; deploy verification pending)** | Scaled in one batch (P11) after gating condition 1 was resolved by the 2026-08-28 no-app-cap decision. No per-pod queue, no naming dependency, no unguarded global state. No resource limits (deliberate: cAdvisor exposes no per-container series to size them from, and measured footprints are 16-69MiB against ~240GB free) |
| flow | **done (P12; deploy verification pending)** | Has a global mutex map, but it wraps redsync (Redis distributed locks) — replica-safe. Scaled together with hook in P12 |
| hook | **done (P12; deploy verification pending)** | Caddy naming blocker resolved by P10: monorepo-etc `infra-caddy/config/Caddyfile` now resolves the Compose service name (`hook-manager`) via a `dynamic a` upstream instead of the fixed `voipbin-hook-manager` container name, so `container_name` removal no longer breaks routing. Scaled in P12. Note: 별도 network-alias 추가(로컬 계획의 P9)는 불필요 판정 — Compose 서비스명이 production 네트워크 DNS alias로 자동 등록됨을 P5/P7/P10에서 실증 |
| api | **blocked** (one remaining blocker) | Caddy naming blocker (i) resolved by P10 — Caddyfile now resolves `api-manager` via `dynamic a`. Remaining: (ii) per-pod `streamData` process-local state needs a code redesign; the Caddy fix alone does not make api scalable. The AudioSocket advertise-address defect is already fixed (`internal/nethandler.AdvertiseIP()`, PR #1209) |
| pipecat | **done** | PR #1230/#171, live-verified 2026-08-30. 본체+사이드카 페어를 4개 named service로 수동 복제, deploy.replicas 미사용. POD_IP는 각 서비스 고유 리터럴(pipecat-manager-1/-2), production 네트워크 alias로 기존 dns_sd 이름(pipecat-manager) 유지 — 코드 변경 없음. 배포 직후 orphan 컨테이너(구 단일 인스턴스) 잔존을 발견해 제거, dns_sd 타겟 정확히 2개로 재확인. per-pod 큐 분리(consumer 1개씩) 및 두 pair 배타적 격리 통화 테스트(양쪽 모두 정상) 완료 |
| transcribe | **done (P18; deploy verification pending)** | Per-pod routing gap resolved by P17 (per-pod RPC routing redesign), unblocking the scale-out. POD_IP is a dead config field (parsed, never consumed) so no value change was needed; HostID stays a random UUID, not POD_IP |
| tts | **done (P19a; deploy verification pending)** | POD_IP now resolves the sidecar's Compose service name (`tts-manager-http`), not its container name — no naming dependency remains. Sidecar stays single-instance |
| timeline | **done** | PR #1236/#174, live-verified 2026-09-01. The intake channel (`pkg/subscribehandler`, 1000-slot buffered channel) drops events silently when full. P8a added drop/occupancy metrics; the 24h single-replica baseline gate is met — `increase(timeline_manager_subscribe_event_dropped_total[24h]) == 0` AND `histogram_quantile(0.99, rate(timeline_manager_subscribe_event_channel_usage_bucket[24h])) == 0.052` (well under the 0.5 threshold). Scaled via the standard recipe (container_name removed, deploy.replicas:2) — no code changes needed. Deployed cleanly with no orphan container; 2/2 containers healthy, queue consumers=2 (subscribe + request), dns_sd targets=2, no active ReplicaDegraded alert |
| route | **blocked** | Global provider healthcheck loop has no cross-replica lock — 2 replicas would double-probe every SIP provider |
| contact | **blocked** | In-process lock needs a Redis distributed-lock (or equivalent) decision first |
| rag | **done** | PR #1234/#173, live-verified 2026-09-01. Already replica-safe: `DocumentClaimForProcessing` claims work via an atomic CAS UPDATE (same class of safety as schedule's claim machinery). Its own unbounded PostgreSQL pool (it bypasses `databasehandler.Connect()`) is covered by the same 2026-08-28 no-app-cap decision that resolved gating condition 1 — no pool-cap fix is planned. Scaled via the standard recipe (container_name removed, deploy.replicas:2) — no code changes needed. Deployed cleanly with no orphan container (unlike pipecat, the Compose service key stayed `rag-manager`, so Compose recreated it in place); 2/2 containers healthy, queue consumer=2, dns_sd targets=2, no active ReplicaDegraded alert |

## References

- [docs/workflows/manager-replica-scaling.md](../workflows/manager-replica-scaling.md) — the durable recipe
- [PR #1212](https://github.com/voipbin/monorepo/pull/1212) — the pilot (schedule-manager)
- [bin-schedule-manager/docs/operations.md](../../bin-schedule-manager/docs/operations.md) — pilot's service-specific notes (backup-path safety)
- [docs/patterns/per-pod-queues.md](../patterns/per-pod-queues.md) — per-pod queue routing background (pipecat's per-pod routing was scaled in P15+16 by giving each replica its own POD_IP literal; transcribe's per-pod routing gap was resolved by P17)
- [2026-08-18-bin-manager-komodo-rollout-tier1-design.md](2026-08-18-bin-manager-komodo-rollout-tier1-design.md), [-tier2-](2026-08-18-bin-manager-komodo-rollout-tier2-design.md) — the Komodo migration this builds on
