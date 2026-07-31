# Dependencies: bin-scheduler-manager

> Dependencies extracted from source by `docs/reference/extractor.sh`. Inbound callers are manually maintained.
> Regenerate outbound data: `bash docs/reference/extractor.sh bin-scheduler-manager`

## Inbound Callers

Services that send RPC requests to this service's queue:

- `bin-scheduler-manager` (self-RPC: the dispatch engine fires the seeded `execution-retention` and `database-backup` schedules at `/v1/executions/prune` and `/v1/backups` on its own request queue)

No external in-repo callers at merge time. Planned consumers of the `SchedulerV1*` client methods in `bin-common-handler/pkg/requesthandler`: the VOIP-1280 doctor checks (next ticket) and `bin-api-manager` (Phase 3 API exposure).

## Outbound RPC Targets

Services this service calls directly (from `go.mod` replace directives):

- `monorepo/bin-agent-manager` (local: `../bin-agent-manager`)
- `monorepo/bin-ai-manager` (local: `../bin-ai-manager`)
- `monorepo/bin-billing-manager` (local: `../bin-billing-manager`)
- `monorepo/bin-call-manager` (local: `../bin-call-manager`)
- `monorepo/bin-campaign-manager` (local: `../bin-campaign-manager`)
- `monorepo/bin-common-handler` (local: `../bin-common-handler`)
- `monorepo/bin-conference-manager` (local: `../bin-conference-manager`)
- `monorepo/bin-contact-manager` (local: `../bin-contact-manager`)
- `monorepo/bin-conversation-manager` (local: `../bin-conversation-manager`)
- `monorepo/bin-customer-manager` (local: `../bin-customer-manager`)
- `monorepo/bin-direct-manager` (local: `../bin-direct-manager`)
- `monorepo/bin-email-manager` (local: `../bin-email-manager`)
- `monorepo/bin-flow-manager` (local: `../bin-flow-manager`)
- `monorepo/bin-hook-manager` (local: `../bin-hook-manager`)
- `monorepo/bin-message-manager` (local: `../bin-message-manager`)
- `monorepo/bin-number-manager` (local: `../bin-number-manager`)
- `monorepo/bin-outdial-manager` (local: `../bin-outdial-manager`)
- `monorepo/bin-pipecat-manager` (local: `../bin-pipecat-manager`)
- `monorepo/bin-queue-manager` (local: `../bin-queue-manager`)
- `monorepo/bin-rag-manager` (local: `../bin-rag-manager`)
- `monorepo/bin-registrar-manager` (local: `../bin-registrar-manager`)
- `monorepo/bin-route-manager` (local: `../bin-route-manager`)
- `monorepo/bin-scheduler-manager` (local: `../bin-scheduler-manager`)
- `monorepo/bin-storage-manager` (local: `../bin-storage-manager`)
- `monorepo/bin-tag-manager` (local: `../bin-tag-manager`)
- `monorepo/bin-talk-manager` (local: `../bin-talk-manager`)
- `monorepo/bin-timeline-manager` (local: `../bin-timeline-manager`)
- `monorepo/bin-transcribe-manager` (local: `../bin-transcribe-manager`)
- `monorepo/bin-transfer-manager` (local: `../bin-transfer-manager`)
- `monorepo/bin-tts-manager` (local: `../bin-tts-manager`)
- `monorepo/bin-webchat-manager` (local: `../bin-webchat-manager`)
- `monorepo/bin-webhook-manager` (local: `../bin-webhook-manager`)

Runtime dispatch note: the DB-driven `target_queue` means the engine can address any request queue in the `commonoutline` allowlist. The only seeded external target in Phase 1 is `bin-number-manager` (`number-renew`); the other two seeds are self-RPC.

## Events Subscribed

RabbitMQ queues this service consumes (from `cmd/*/main.go` subscribeTargets):

- `bin-manager.customer-manager.event`

## Events Published

Webhook events this service publishes (from `PublishWebhookEvent` calls in source):

_No published events detected._

(Internal-only events — `schedule_created/updated/deleted`, `execution_succeeded/failed` — are published via notifyhandler `PublishEvent` on `bin-manager.scheduler-manager.event`; every Phase 1 schedule is nil-customer, so no customer webhooks exist. See `docs/architecture.md`.)

## WebhookMessage Contracts

Field-level schemas for entities this service exposes are defined in the RST docs:
→ `bin-api-manager/docsdev/source/` — do not restate field lists here. (Phase 1 has no WebhookMessage and no RST surface; this arrives with Phase 3.)
