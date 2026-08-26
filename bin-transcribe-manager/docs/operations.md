# bin-transcribe-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Session stuck in `progressing` | `call_hangup` event not received or not processed. **Also**: the zombie-session invariant (see [Zombie-Session Invariant and Recovery](#zombie-session-invariant-and-recovery) below) deliberately refuses to move a transcribe to `done` while any of its streaming sessions genuinely failed to stop | Check subscribe handler queue; manually stop via `POST /v1/transcribes/{id}/stop`. See the section below for the full mitigation picture, including a current limitation that prevents automatic retry today |
| STT RPC routed to wrong pod | Stale `host_id` after pod restart (Calico POD_IP recycle) | See [per-pod-queues.md](../../docs/patterns/per-pod-queues.md) for known limitation; session must be recreated |
| No transcripts appearing | WebSocket to Asterisk not established | Check `streaming_handler` logs for dial errors; verify `MediaURI` from `ExternalMediaStart` |
| GCP auth failure | ADC not configured | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid mounted service account key file |
| AWS auth failure | Missing credentials | Verify `AWS_ACCESS_KEY` / `AWS_SECRET_KEY` env vars |
| Session health-check failing | Pod hosting session is down | Session is lost; streaming cannot be resumed — client must recreate |
| `customer_deleted` cascade not running | Subscribe handler not consuming customer-manager events | Check queue binding: `bin-manager.customer-manager.event` |
| Transcribe start/stop returns a `STT_NOT_CONFIGURED` error (`*cerrors.VoipbinError`, status `Unavailable`) | Neither GCP nor AWS STT client initialized at startup, so the service is running with the disabled streaming handler. Common causes: an invalid or placeholder `GOOGLE_APPLICATION_CREDENTIALS` key file plus no AWS credentials | Fix the GCP key file or set `AWS_ACCESS_KEY` / `AWS_SECRET_KEY`, then restart the service. Confirm with the provider-initialization log check below — a healthy boot logs `GCP STT provider initialized` and/or `AWS STT provider initialized`; a degraded boot logs `No STT providers available. Streaming transcribe is disabled until GCP or AWS credentials are configured.` |

## Zombie-Session Invariant and Recovery

`pkg/transcribehandler/stop.go`'s zombie-session invariant deliberately refuses to move a transcribe to `done` while any of its streaming sessions genuinely failed to stop (e.g. a transient call-manager RPC failure). This correctly prevents an orphaned live session from being marked done, but leaves the transcribe in `progressing` until the stop actually succeeds.

Current mitigations, and their limits:

- `streamingHandler.Stop` removes the entry from the in-memory session map as soon as the external media is actually stopped, so a retried stop never re-attempts an already-stopped streaming.
- `pkg/transcribehandler/health.go`'s `stopOrReschedule` will requeue another health check (up to `defaultStopRescheduleMaxRetryCount` attempts) when `Stop` fails with the specific `STREAMING_STOP_FAILED` error — any other failure (e.g. an invalid reference type) is treated as permanent and is not retried.
- **This reschedule mechanism is currently dormant.** Nothing in the codebase schedules the *first* health check for a transcribe: `startLive` never calls `TranscribeV1TranscribeHealthCheck`, and no other service, `cmd`, or subscribe handler does either. The only callers of that RPC are `health.go`'s own reschedule calls. In practice, the periodic health check does **not** currently retry a stuck stop on its own — bootstrapping the first health check (e.g. scheduling one from `startLive`) is a separate follow-up that has not been implemented yet.
- `Delete` on a non-`done` transcribe still proceeds even if `Stop` fails, so a stuck session never blocks the customer-deletion cascade — but this only means the record is removed; the streaming session itself may still be live on Asterisk with no VoIPbin record left to stop it (see the log line `Could not stop the transcribe before delete; streaming may still be active on Asterisk...`).

**Until the health-check bootstrap lands, the only real recovery paths for a transcribe stuck in `progressing` after a genuine stop failure are:**
1. A manual `POST /v1/transcribes/{id}/stop` retry.
2. `Delete`, which proceeds regardless of stop failure (see above) — but does not itself confirm the streaming session was actually stopped, so treat it as record cleanup, not confirmed session teardown.

**These recovery paths, and the manual-retry framing above, assume this service is running with `replicas: 1` (see `k8s/deployment.yml`).** `pkg/transcribehandler/stop.go`'s `isSafeToConsiderStopped` treats a `NotFound` from the in-memory session lookup as proof the session cannot possibly still be alive — which is only true when a single pod owns every session. `TranscribeV1TranscribeStop` and `TranscribeV1TranscribeHealthCheck` are both routed through the shared `QueueNameTranscribeRequest` queue rather than a per-pod queue, so if `replicas` is ever raised above 1 without first fixing that routing gap, a "manual stop RPC retry" against a non-owning pod would return `NotFound` and be treated as a confirmed stop instead of a recovery attempt — silently completing the zombie session rather than recovering it. See the comment on `isSafeToConsiderStopped` for the full explanation.

## Debugging Guide

**Check active streaming sessions (log search):**
```bash
kubectl logs -n voipbin -l app=transcribe-manager --tail=200 | grep -E "streaming|WebSocket|ERROR"
```

**Check subscribe queue processing:**
```bash
kubectl exec -n voipbin deploy/rabbitmq -- rabbitmqctl list_queues name messages | grep -E "transcribe|customer"
```

**Check transcription session status via DB:**
```bash
./bin/transcribe-control transcribe get --id <uuid>
./bin/transcribe-control transcribe list --customer_id <uuid>
```

**Manually stop a stuck session:**
```bash
./bin/transcribe-control transcribe stop --id <uuid>
```

**Verify provider initialization at startup:**
```bash
kubectl logs -n voipbin -l app=transcribe-manager | grep -E "provider|GCP|AWS|initialized"
```

**Check ARI (Asterisk) event processing:**
```bash
kubectl logs -n voipbin -l app=transcribe-manager --tail=200 | grep -E "ari_event|ExternalMedia"
```

## Configuration

Service uses Cobra and Viper (`internal/config/main.go`). Configuration loaded once via `PersistentPreRunE` hook, accessed globally via `config.Get()`.

| Flag / Env Var | Description | Default |
|----------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server URL | required |
| `redis_address` / `REDIS_ADDRESS` | Redis server address | required |
| `redis_password` / `REDIS_PASSWORD` | Redis authentication | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `aws_access_key` / `AWS_ACCESS_KEY` | AWS Transcribe access key | optional (if GCP configured) |
| `aws_secret_key` / `AWS_SECRET_KEY` | AWS Transcribe secret key | optional (if GCP configured) |
| `pod_ip` / `POD_IP` | Pod IP (Kubernetes Downward API) — used as `HostID` for per-pod queue | required |
| `streaming_listen_port` / `STREAMING_LISTEN_PORT` | Port for WebSocket streaming connections | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

GCP authentication uses Application Default Credentials. Neither STT provider is strictly required to boot: if neither initializes, the service starts with streaming transcribe disabled (see the failure-modes table above). Configure at least one provider for a working streaming transcribe path.

## Prometheus Metrics

Metrics registered in handler `init()` functions, exposed at `PROMETHEUS_LISTEN_ADDRESS` (default `:2112/metrics`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `transcribe_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `transcribe_manager_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event subscription processing duration |
| `transcribe_manager_ari_event_listen_process_time` | Histogram | — | ARI event processing duration |
| `transcribe_manager_ari_event_listen_total` | Counter | — | Total ARI events received |
| `transcribe_manager_transcribe_create_total` | Counter | `type` | Transcription sessions created (by provider) |
| `transcribe_manager_transcript_transcript_create_total` | Counter | — | Transcript segments created |

## Deployment (Komodo)

Komodo-managed (VOIP-1360, Tier 5 - third of the 4 GCP-credential-file
services, same pattern as bin-storage-manager/VOIP-1358 and
bin-rag-manager/VOIP-1359). Deployed via
`.circleci/scripts/render-image-tag.sh` + `.circleci/scripts/komodo-api-deploy.sh`
from `komodo/docker-compose.yml`.

**GCP credential file**: same Docker Compose environment-sourced `secrets:`
block as bin-storage-manager. `GCP_SA_JSON` comes from
`komodo/environment.env`, passed as `komodo-api-deploy.sh`'s 3rd argument.

**`POD_IP`**: not a real Komodo Variable, same as bin-pipecat-manager.
`install/`'s own `docker-compose.yml.dist` already falls back to a
literal (`${POD_IP:-127.0.0.1}`) since the K8s Downward API wiring this
was meant for was never carried over to Docker Compose - the Komodo
compose file keeps that same literal, matching current production
behavior exactly. This service's per-pod queue routing
(`bin-manager.transcribe-manager.request.<POD_IP>`, see this service's
CLAUDE.md) is therefore unaffected by the cutover - it already runs as a
single instance with this same fixed value today.

No healthcheck block: distroless (`gcr.io/distroless/static-debian12`),
same as every other distroless `bin-*-manager` service (VOIP-1342 pilot
established the fleet-standard `wget` healthcheck can never pass on this
image).
