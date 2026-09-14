# VOIP-1524: Restore DATABASE_DSN on bin-timeline-manager Komodo deploy

Date: 2026-09-14
Ticket: VOIP-1524
Status: design APPROVED (review loop: 1st Request Changes -> fixed -> 2nd/3rd consecutive APPROVE, 2026-09-14) -> implemented in same branch/PR

## Problem

Production `GET/POST /v1.0/timeline-analyses` (all 4 sub-routes) returns
HTTP 503 SERVICE_UNAVAILABLE immediately (sub-second, not a timeout). The
AI activeflow analysis feature (including the square-admin AI Analysis
panel) has been fully unavailable in production for 11+ days.

## Root cause (verified, not hypothesized)

- Code is correct: `bin-timeline-manager/pkg/listenhandler/v1_analyses.go`
  guards all 4 analysis routes with `if h.analysisHandler == nil { return
  503 }` — the intentional fail-safe added by VOIP-1197 (June 2026).
- `cmd/timeline-manager/main.go` (L193-215) only builds `analysisHandler`
  when `DATABASE_DSN` is non-empty; otherwise it logs
  `"DATABASE_DSN not configured; analysis endpoints are disabled."` and
  serves the ClickHouse-only role.
- Live evidence:
  - Loki (container start, 11 days ago): the exact WARNING above.
  - Komodo `InspectDockerContainer` (both replicas): 9 env vars present,
    `DATABASE_DSN` absent.
- Regression origin: commit 642904b42 (2026-08-18, VOIP-1349 GKE->Komodo
  Tier-3 rollout) authored `bin-timeline-manager/komodo/docker-compose.yml`
  from the stale premise "timeline-manager depends on clickhouse, not
  db/redis - no DATABASE_DSN or REDIS_ADDRESS here" (comment in the file).
  That premise predates VOIP-1197 (2026-06-24), which made timeline-manager
  additionally depend on MySQL for the analysis store. The GKE manifest
  (`k8s/deployment.yml`) had the mapping; the Komodo compose never got it.
  This is the exact "recurrence via infra migration" failure mode documented
  for nil-safe optional handlers: the fail-safe makes the omission quiet
  (no crash-loop, healthy pods, fast 503s), so nothing automated surfaced it.

## Precondition gate (Gate 1) — PASSED via live measurement

Measured directly against production MariaDB (172.24.128.1:3306 over VPN,
2026-09-14):

- `SHOW TABLES LIKE 'timeline_analyses'` -> table exists.
- `DESCRIBE timeline_analyses` -> 9 columns (id, customer_id, activeflow_id,
  status, result, model, error, tm_create, tm_update), matching migration
  a63b82d73655.
- `SELECT version_num FROM alembic_version` -> `79119e39e511`, a revision
  later than a63b82d73655 (no DROP of this table exists in any migration).

No schema work is needed. The fix is ops-config only.

## Fix

One line in `bin-timeline-manager/komodo/docker-compose.yml`:

```yaml
      - DATABASE_DSN=[[BIN_MANAGER__DATABASE_DSN_BIN]]
```

- Reuses the existing Komodo Variable `BIN_MANAGER__DATABASE_DSN_BIN`
  (key verified present in `infra-secret/secrets-source/bin-manager`),
  the same mapping the GKE manifest used (`secret voipbin key
  DATABASE_DSN_BIN` -> env `DATABASE_DSN`) and the same reference 30 other
  bin-* komodo compose files already use. No companion infra-secret PR.
- The stale "no DATABASE_DSN" comment in the file header is corrected in
  the same change so the next migration/refactor does not re-inherit the
  wrong premise.
- `bin-timeline-manager/docs/operations.md` carries the SAME stale premise
  in two places and is corrected in the same PR (Service docs sync rule +
  this design's own recurrence-prevention logic): (1) the Configuration
  table omits `DATABASE_DSN` and `ANALYSIS_MODEL_STAGE1/2/3` entirely —
  rows added, with the 503-when-unset consequence stated; (2) the
  "Deployment (Komodo)" section claims "does not use MySQL/Redis at all"
  and lists Komodo Variables without `DATABASE_DSN` — rewritten to state
  the MySQL analysis-store dependency and the exact variable mapping.

## Explicitly out of scope (parity discipline, anti-overengineering)

- `ANALYSIS_MODEL_STAGE1/2/3`: NOT added. The retired GKE production
  manifest never set them either; production has always run on the
  ai-manager gateway default (`gemini-2.5-flash`, verified fallback in
  `bin-ai-manager/pkg/analysishandler/run.go` L44-51). Adding them now
  would change behavior beyond restoring parity. If stage-tiered models
  are wanted later, that is a separate product decision + ticket.
- `REDIS_ADDRESS`: genuinely not used by timeline-manager; the original
  comment was right about this half.
- Code changes: none. The nil-guard fail-safe works as designed.

## Risk assessment

- Wrong-but-nonempty DSN: `commondatabasehandler.Connect` (includes Ping)
  fails at startup -> `return err` -> container exits -> `restart: always`
  + `replicas: 2` => crash-loop. Probability low: the identical variable
  reference is live in 30 running services.
- Valid DSN + table missing: ruled out by Gate 1 live measurement; and even
  then errors map through `analysisErrorResponse` (429/409/404/500), no
  panic, ClickHouse ingestion path unaffected (separate code path).
- Blast radius: env-only change to one stack; deploy is a container
  recreate. RabbitMQ competing consumers + the second replica cover the
  seconds-level restart window; ClickHouse ingestion resumes on start
  (standard rollout pattern for this fleet).

## Verification plan (post-deploy)

1. Komodo `InspectDockerContainer`: both replicas show `DATABASE_DSN` set.
2. Loki: startup log shows `"Analysis handler initialized (MySQL store +
   requesthandler)."` and the WARNING is gone.
3. Live probe: `GET /v1.0/timeline-analyses?accesskey=...` returns 200 with
   a list payload (empty list acceptable), not 503.
4. Replicas stay Running with 0 restarts (rules out the wrong-DSN
   crash-loop scenario).
