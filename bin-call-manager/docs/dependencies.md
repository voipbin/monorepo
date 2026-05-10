# Dependencies: bin-call-manager

> Auto-generated from `docs/reference/extractor.sh`. Do not edit manually.
> Regenerate: `bash docs/reference/extractor.sh bin-call-manager && bash docs/reference/render-deps.sh bin-call-manager`

## Inbound Callers

Services that send RPC requests to this service's queue:

- `bin-api-manager` (call creation, listing, hangup, media operations)
- `bin-flow-manager` (call actions: play, talk, hold, mute, recording, confbridge)
- `bin-billing-manager` (CDR lookup)

## Outbound RPC Targets

Services this service calls directly (from `go.mod` replace directives):

- `monorepo/bin-agent-manager` (local: `../bin-agent-manager`)
- `monorepo/bin-billing-manager` (local: `../bin-billing-manager`)
- `monorepo/bin-common-handler` (local: `../bin-common-handler`)
- `monorepo/bin-conference-manager` (local: `../bin-conference-manager`)
- `monorepo/bin-conversation-manager` (local: `../bin-conversation-manager`)
- `monorepo/bin-customer-manager` (local: `../bin-customer-manager`)
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
- `monorepo/bin-sentinel-manager` (local: `../bin-sentinel-manager`)
- `monorepo/bin-storage-manager` (local: `../bin-storage-manager`)
- `monorepo/bin-tag-manager` (local: `../bin-tag-manager`)
- `monorepo/bin-timeline-manager` (local: `../bin-timeline-manager`)
- `monorepo/bin-transcribe-manager` (local: `../bin-transcribe-manager`)
- `monorepo/bin-transfer-manager` (local: `../bin-transfer-manager`)
- `monorepo/bin-tts-manager` (local: `../bin-tts-manager`)
- `monorepo/bin-webhook-manager` (local: `../bin-webhook-manager`)

## Events Subscribed

RabbitMQ queues this service consumes (from `cmd/*/main.go` subscribeTargets):

- `asterisk.all.event`
- `bin-manager.customer-manager.event`
- `bin-manager.flow-manager.event`
- `bin-manager.sentinel-manager.event`

## Events Published

Webhook events this service publishes (from `PublishWebhookEvent` calls in source):

- `call.EventTypeCallCreated`
- `call.EventTypeCallDeleted`
- `call.EventTypeCallHangup`
- `call.EventTypeCallUpdated`

## WebhookMessage Contracts

Field-level schemas for entities this service exposes are defined in the RST docs:
→ `bin-api-manager/docsdev/source/` — do not restate field lists here.
