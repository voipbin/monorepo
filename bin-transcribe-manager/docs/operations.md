# bin-transcribe-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Session stuck in `progressing` | `call_hangup` event not received or not processed. **Also**: the zombie-session invariant (see [Zombie-Session Invariant and Recovery](#zombie-session-invariant-and-recovery) below) deliberately refuses to move a transcribe to `done` while any of its streaming sessions genuinely failed to stop | Check subscribe handler queue; manually stop via `POST /v1/transcribes/{id}/stop`. See the section below for the full mitigation picture, including a current limitation that prevents automatic retry today |
| STT RPC routed to wrong/nonexistent pod | Stale `host_id` after any process restart — `host_id` is a random UUID generated fresh at startup (not derived from `POD_IP` or pod identity), so it changes every time the process restarts, not only when the pod IP changes | Session must be recreated; see [docs/architecture.md](architecture.md) for per-pod queue naming |
| No transcripts appearing | WebSocket to Asterisk not established | Check `streaming_handler` logs for dial errors; verify `MediaURI` from `ExternalMediaStart` |
| GCP auth failure | ADC not configured | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid mounted service account key file |
| AWS auth failure | Missing or wrong credentials | Check the start log of the replica: `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` must both be set (an unresolved placeholder or only one of the two means AWS is not available). See "AWS credential" below |
| Session health-check failing | Pod hosting session is down | Session is lost; streaming cannot be resumed — client must recreate |
| `customer_deleted` cascade not running | Subscribe handler not consuming customer-manager events | Check queue binding: `bin-manager.customer-manager.event` |
| Transcribe start returns a `STT_NOT_CONFIGURED` error (`*cerrors.VoipbinError`, status `Unavailable`) | No STT provider is available when the request arrives: there is no GCP client (an invalid or placeholder `GOOGLE_APPLICATION_CREDENTIALS` key file) and no AWS pair in the environment. Nothing is created for the request. Stop of an unknown session is `STREAMING_NOT_FOUND`, which the transcribe handler treats as already stopped | Fix the GCP key file, or set both AWS variables (see "AWS credential" below); the replicas pick up a changed pair when they restart. The start up log says `GCP STT provider initialized` and/or `AWS STT provider initialized`, or `No STT provider is available now.` |

## Schema Migration Failures

Migrations run at startup, before the service listens. A failure exits the process with a non-zero code.

| Symptom (log) | Meaning | Action |
|---|---|---|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. `transcribe-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so, then start again. See the migration section of `CLAUDE.md` |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | Restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | Only for ALTER style migrations after the baseline (the baseline does not block on an existing table): a long transaction holds the table; the migration left the database dirty | Find the long transaction, then force `N-1` and restart |
| `CREATE command denied ... transcribe_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during rollback; no action |

`transcribe-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table if missing).

## Order on a New Database

On a new database the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first. The service account also needs permission to create tables (for the version table). Create the database with `CHARACTER SET utf8 COLLATE utf8_general_ci`, like the install scripts do.

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

**These recovery paths are safe regardless of replica count.** `pkg/transcribehandler/stop.go`'s `isSafeToConsiderStopped` treats a `NotFound` from the in-memory session lookup as proof the session cannot possibly still be alive. `TranscribeV1TranscribeStop` and `TranscribeV1TranscribeHealthCheck` are now both routed directly to the owning pod's per-pod queue (`bin-manager.transcribe-manager-<host_id>.request`, keyed off the transcribe's `HostID`) rather than the shared `QueueNameTranscribeRequest` queue, so a "manual stop RPC retry" always reaches the one pod that could possibly hold the session — a `NotFound` response reliably means that pod has no session for this transcribe, i.e. it is already stopped, independent of how many other replicas exist. See the comment on `isSafeToConsiderStopped` for the full explanation.

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
| (no flag) / `AWS_ACCESS_KEY` | AWS Transcribe access key. Environment only, never a flag; read at the time of use. Not set, empty or an unresolved `[[...]]` placeholder means not configured | optional |
| (no flag) / `AWS_SECRET_KEY` | AWS Transcribe secret key. Same rules as `AWS_ACCESS_KEY`; the two are used only together | optional |
| `transcribe_config_encryption_keys` / `TRANSCRIBE_CONFIG_ENCRYPTION_KEYS` | Not read by the code. The deployment definition still passes it; a later change removes the line | optional |
| `pod_ip` / `POD_IP` | Pod IP (Kubernetes Downward API); config field is intended for the AudioSocket streaming listener bind address — **not** the per-pod queue `host_id`, which is a random UUID generated at process startup independent of `POD_IP` | required |
| `streaming_listen_port` / `STREAMING_LISTEN_PORT` | Port for WebSocket streaming connections | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

GCP authentication uses Application Default Credentials. Neither STT provider is strictly required to boot: if none is available, the service starts and every request to start a streaming is answered `STT_NOT_CONFIGURED` (see the failure-modes table above). Configure at least one provider for a working streaming transcribe path.

## AWS credential

The AWS credential is read from two environment variables, and only from there: `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` (the same variables and values as `bin-tts-manager`). Nothing is stored in the database and no admin API or command writes them.

| Variable | Used by |
|----------|---------|
| `AWS_ACCESS_KEY` | AWS Transcribe streaming |
| `AWS_SECRET_KEY` | AWS Transcribe streaming |

The two variables are one credential and are used only together. A variable that is not set, is empty or still holds an unresolved deployment placeholder (a value that starts with `[[`) means that it is not configured; a pair with one variable missing is "AWS not available" (an empty pair and no error, so a deployment with GCP goes on without AWS and one with no GCP refuses the start with `STT_NOT_CONFIGURED`). At start the service logs one line per variable (name and state only, never a value): an info line for a variable that is not set (AWS is optional), a warning with the words "unresolved placeholder" for a placeholder, and one more warning when exactly one of the two is set. Nothing is logged per call. The pair is read for every transcribe start and the AWS client is built again only when the pair changed; an environment changes with a new deployment only, so a rotation takes effect when the replicas restart. A stream that is already running keeps the client it started with.

### Rotation

1. Create the new credential at AWS. Keep the old one valid.
2. Change `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` together in the deployment variable store and redeploy the service. During a rolling deploy two replicas hold different pairs until the deploy ends.
3. Check the start log of each replica (both variables set, no placeholder warning), then one transcribe start that uses AWS (a start with `provider` `aws`). If AWS refuses the credential, correct the variables and redeploy.
4. Revoke the old credential at AWS after the check. Keep it valid until the rollback window has ended: a rollback is the previous image together with the previous deployment definition.

Shared variables: `bin-tts-manager` reads the same two variables, so a rotation changes both services. The values must be equal in both; this is checked in the deployment variables, not in the code. Redeploy both services before the old credential is revoked, check a transcribe start and a TTS call after the change, and revoke the old credential last.

Recovery: a variable that holds an unresolved placeholder or only one variable set shows up in the start log; fix the deployment variable and redeploy.

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
behavior exactly. Note that this service's per-pod queue routing does
not actually key off `POD_IP` at all - the per-pod queue name is
`bin-manager.transcribe-manager-<host_id>.request`, where `<host_id>`
is a random UUID generated fresh at each process start (see this
service's CLAUDE.md), so the queue name already changes on every
restart regardless of `POD_IP`. The Komodo cutover is therefore
unaffected either way - it already runs as a single instance today.

No healthcheck block: distroless (`gcr.io/distroless/static-debian12`),
same as every other distroless `bin-*-manager` service (VOIP-1342 pilot
established the fleet-standard `wget` healthcheck can never pass on this
image).
