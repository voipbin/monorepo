# bin-transcribe-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Session stuck in `progressing` | `call_hangup` event not received or not processed. **Also**: the zombie-session invariant (see [Zombie-Session Invariant and Recovery](#zombie-session-invariant-and-recovery) below) deliberately refuses to move a transcribe to `done` while any of its streaming sessions genuinely failed to stop | Check subscribe handler queue; manually stop via `POST /v1/transcribes/{id}/stop`. See the section below for the full mitigation picture, including a current limitation that prevents automatic retry today |
| STT RPC routed to wrong/nonexistent pod | Stale `host_id` after any process restart — `host_id` is a random UUID generated fresh at startup (not derived from `POD_IP` or pod identity), so it changes every time the process restarts, not only when the pod IP changes | Session must be recreated; see [docs/architecture.md](architecture.md) for per-pod queue naming |
| No transcripts appearing | WebSocket to Asterisk not established | Check `streaming_handler` logs for dial errors; verify `MediaURI` from `ExternalMediaStart` |
| GCP auth failure | ADC not configured | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid mounted service account key file |
| AWS auth failure | Missing or wrong credentials | Check `./transcribe-control config list` (source and has_secret of both keys) (the `AWS_ACCESS_KEY` / `AWS_SECRET_KEY` env vars are not read any more). See "AWS credential" below |
| Session health-check failing | Pod hosting session is down | Session is lost; streaming cannot be resumed — client must recreate |
| `customer_deleted` cascade not running | Subscribe handler not consuming customer-manager events | Check queue binding: `bin-manager.customer-manager.event` |
| Transcribe start returns a `STT_NOT_CONFIGURED` error (`*cerrors.VoipbinError`, status `Unavailable`) | No STT provider is available when the request arrives: there is no GCP client (an invalid or placeholder `GOOGLE_APPLICATION_CREDENTIALS` key file) and no stored AWS credential pair. Nothing is created for the request. Stop of an unknown session is `STREAMING_NOT_FOUND`, which the transcribe handler treats as already stopped | Fix the GCP key file, or store the AWS pair (see "AWS credential" below). No restart is needed for a stored or changed AWS pair. The start up log says `GCP STT provider initialized` and/or `AWS STT provider initialized`, or `No STT provider is available now.` |
| Transcribe start returns `STT_KEY_UNAVAILABLE` (status `Unavailable`) | There is no GCP client and the AWS credential could not be read: the database is down, a stored value can not be decrypted, or the master key list is missing while a row exists. | Restore the database or the master key list (`TRANSCRIBE_CONFIG_ENCRYPTION_KEYS`), or clear both stored keys and store the pair again. A deployment that has GCP is not affected: the request goes on without AWS and logs one warning |

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
| `aws_access_key` / `AWS_ACCESS_KEY` | AWS Transcribe access key (removed: the service does not read it any more; store `aws.access_key` with `transcribe-control config set`) | not read |
| `aws_secret_key` / `AWS_SECRET_KEY` | AWS Transcribe secret key (removed: the service does not read it any more; store `aws.secret_key` with `transcribe-control config set`) | not read |
| `transcribe_config_encryption_keys` / `TRANSCRIBE_CONFIG_ENCRYPTION_KEYS` | Master key list that encrypts the stored AWS credential (environment only, never a flag, never logged). Without it the service starts, a stored secret is reported as undecryptable, and storing a secret is refused | optional |
| `pod_ip` / `POD_IP` | Pod IP (Kubernetes Downward API); config field is intended for the AudioSocket streaming listener bind address — **not** the per-pod queue `host_id`, which is a random UUID generated at process startup independent of `POD_IP` | required |
| `streaming_listen_port` / `STREAMING_LISTEN_PORT` | Port for WebSocket streaming connections | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

GCP authentication uses Application Default Credentials. Neither STT provider is strictly required to boot: if none is available, the service starts and every request to start a streaming is answered `STT_NOT_CONFIGURED` (see the failure-modes table above). Configure at least one provider for a working streaming transcribe path.

## AWS credential

The AWS credential is two stored settings, read from the database only. A key without a stored row is not configured and AWS is not available; the environment variables `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` are not read by this service any more (VOIP-1721). Enter the pair with `transcribe-control config set` or the admin API; `config clear` removes a key. The start logs an info line for each key that is not configured (key name only), because AWS is an optional provider.

| Stored key | Used by |
|------------|---------|
| `aws.access_key` | AWS Transcribe streaming |
| `aws.secret_key` | AWS Transcribe streaming |

The two keys are one credential and are used only together: both stored. A pair with one key missing is "AWS not available", and a stored pair with one key cleared is not replaced by anything. The pair is read for every transcribe start (one bounded database read of at most 2 seconds, none for stop), and the AWS client is built again only when the pair changed, so a change is used by the next start of every replica with no restart. A stream that is already running keeps the client it started with.

```bash
# the value is read from stdin only (one line; a key is at most 512 bytes) and is never printed
printf '%s\n' "$NEW_ACCESS_KEY" | ./transcribe-control config set --key aws.access_key --actor-id <uuid>
printf '%s\n' "$NEW_SECRET_KEY" | ./transcribe-control config set --key aws.secret_key --actor-id <uuid>
./transcribe-control config list
./transcribe-control config clear --key aws.access_key --actor-id <uuid>
./transcribe-control config clear --key aws.secret_key --actor-id <uuid>
```

The same two keys can also be stored, deleted and read through the admin API of api-manager (`/service_configs/transcribe`, project super admin only), which reaches this service as `/v1/configs` requests. The history records the source `api` for the admin API and `cli` for the command line. The request body of a secret is never logged, and a request that looks like a config request is not routed to any other handler. The master key list is `TRANSCRIBE_CONFIG_ENCRYPTION_KEYS` (environment only, passed by the deployment definition); without it `config list` and `config clear` work and `config set` is refused. The `config` commands need the database and the master key list only.

Entering the pair or rotating it takes two `config set` commands, so run them back to back. After the first command of a first entry the pair is incomplete and AWS is not available (a deployment with no GCP refuses a transcribe start in that time). After the first command of a rotation the pair is mixed (one new stored key, one old stored key or none) and the next start does not use a mixed pair, so AWS is not available until the second command ends. An image from before VOIP-1721 reads `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` from the environment again when a key has no stored row; a credential rotated in the database only is not rotated for such an image. Keep the old credential valid until the rollback window has ended. For such an image, the deployment does not inject the variables any more (VOIP-1730), so a rollback to such an image first restores those lines in the deployment definitions (revert of that change); this change does not delete the variables or secret keys of the deployment platform, so a restored line supplies the value again.

Rotation:
1. Create the new credential at AWS. Keep the old one valid.
2. Make sure every replica runs the image of this feature. Do not enter a value while a rolling deploy is half done, with the command line or the admin API: an older replica ignores the stored value (and does not serve the admin API), so the replicas would use different credentials.
3. Run the two `config set` commands, back to back.
4. Check one transcribe start that uses AWS (a start with `provider` `aws`). If AWS refuses the credential, set the right values (`config clear` leaves AWS not configured).
5. Revoke the old credential at AWS after the check. A rollback of the image (see above) can use the environment pair again only after the lines of the deployment definitions are restored (see above), so keep the old credential valid until the check passed.

Dual input: bin-tts-manager stores the same kind of credential in its own settings. Each service has its own stored settings, so the source is two places and nothing keeps them equal: rotating in one service only leaves the other on the old credential, and revoking the old credential then breaks that service. Rotate both services before the old credential is revoked, check a transcribe start and a TTS call after the change, and revoke the old credential last.

Recovery: a stored value that can not be decrypted (the master key is missing, mistyped or changed) is not replaced by an empty pair. A transcribe start answers `STT_KEY_UNAVAILABLE` when GCP is not available, `config list` shows `db_undecryptable`, and the start up logs an error line. Restore the master key list, or run both `config clear` commands and store the pair again.

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
