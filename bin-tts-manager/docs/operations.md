# bin-tts-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Service exits at startup ("Could not create the tts handler") | GCP ADC credentials not available — the daemon fails fast instead of nil-panicking on the first `/v1/speeches` request | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid credential file (`/run/secrets/google_service_account.json`, materialized by the Komodo compose `secrets:` block); verify the `BIN_MANAGER__GOOGLE_APPLICATION_CREDENTIALS_JSON` Komodo Variable is synced |
| Batch TTS returns no audio URL | GCP request failing at runtime (quota, endpoint), or `/shared-data` not writable (volume missing from the compose stack) | Check `speech_fallback_total` metric (AWS Polly is the fallback provider); verify the `shared-data` volume is mounted in `komodo/docker-compose.yml` |
| Call connects but TTS audio is silent while `/v1/speeches` succeeds | Asterisk cannot fetch the media URL — `voipbin-tts-manager-http` sidecar down, or `POD_IP` missing/wrong so the URL host does not resolve | Check the `tts-manager-http` container is running on the `production` network; verify `POD_IP=tts-manager-http` in the compose stack |
| AWS Polly fallback failing | Missing AWS variables, only one of the two AWS variables set, or an unresolved placeholder | Check the start log (see [Provider keys](#provider-keys)): `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` must both be set; check AWS Polly quotas |
| Streaming session RPC timeout | RPC routed to wrong pod (wrong per-pod queue) | Verify `host_id` matches `HOSTNAME` of target pod; check per-pod queue binding |
| AudioSocket connection refused | Go service port 8080 not listening | Check pod readiness; verify no port conflict with Python sidecar |
| Audio file not served by sidecar | `/shared-data` volume not mounted or empty | Check pod volume mount; verify Go service wrote the file before sidecar serves it |
| ElevenLabs WebSocket disconnect | Rate limit or API key invalid | Check `streaming_error_total` metric; verify the `ELEVENLABS_API_KEY` variable (the start log shows its state, never the value) |
| Keep-alive timeout | Network issue between pod and ElevenLabs | Check `streaming_error_total`; session is cleaned up automatically |
| `POD_IP` not set | GKE: missing Downward API configuration. Compose: env dropped from the stack | GKE: verify `k8s/deployment.yml` injects `status.podIP` as `POD_IP`. Compose: verify `POD_IP=tts-manager-http` in `komodo/docker-compose.yml` |

## Schema Migration Failures

Migrations run at startup, before the service connects the cache and the message broker. A failure exits the process with a non-zero code and the container is restarted.

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. Run `tts-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so, then start again. The image has no shell, so run the command in a one-off container of the same image: `docker run --rm --entrypoint ./tts-control -e DATABASE_DSN=<dsn> <image> migrate force <N>` (the container must reach the database; quote a DSN that holds special characters) |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | A long transaction holds a metadata lock on the table (even `CREATE TABLE IF NOT EXISTS` takes it briefly) | Find the long transaction, then restart. If the database is marked dirty, follow the dirty row |
| `CREATE command denied ... tts_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during a rollback; no action |

`tts-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table when it is missing).

On a new database the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first. The service account also needs permission to create tables (for the version table). The database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`, like the install scripts do.

## Debugging Guide

**Check active streaming sessions (via metrics):**
```bash
kubectl exec -n voipbin -l app=tts-manager -- curl -s localhost:2112/metrics | grep streaming_active
```

**Check batch TTS creation rate:**
```bash
kubectl exec -n voipbin -l app=tts-manager -- curl -s localhost:2112/metrics | grep speech_request_total
```

**Check provider fallback rate:**
```bash
kubectl exec -n voipbin -l app=tts-manager -- curl -s localhost:2112/metrics | grep speech_fallback_total
```

**Check audio file presence on shared volume:**
```bash
kubectl exec -n voipbin -l app=tts-manager -c tts-manager -- ls /shared-data/
```

**Check per-pod queue binding:**
```bash
kubectl exec -n voipbin deploy/rabbitmq -- rabbitmqctl list_queues name messages | grep tts-manager
```

**Service logs for streaming errors:**
```bash
kubectl logs -n voipbin -l app=tts-manager -c tts-manager --tail=200 | grep -E "ERROR|streaming|elevenlabs"
```

## Deployment

bin-tts-manager deploys via Komodo (VOIP-1348 Tier 2 rollout, following
the VOIP-1342/bin-call-manager pilot and VOIP-1347/Tier 1 pattern)
instead of the older SSH + `versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-tts-manager/komodo/docker-compose.yml` (git
  is the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-tts-manager-deploy` job's poll/running checks.
- **GCP credential file:** same Docker Compose environment-sourced
  `secrets:` block as bin-storage-manager/bin-transcribe-manager.
  `GCP_SA_JSON` comes from `komodo/environment.env`, passed as
  `komodo-api-deploy.sh`'s 3rd argument, and is materialized inside the
  container at `/run/secrets/google_service_account.json`
  (`GOOGLE_APPLICATION_CREDENTIALS` points there). tts-manager was missed
  in the original GCP-credential-file rollout — on GKE this worked
  implicitly via workload-identity ADC, so the gap only surfaced on
  <prod-host> (see NOJIRA-Fix-tts-manager-gcp-credentials).
- **Batch-TTS media delivery:** the stack runs two containers sharing a
  named `shared-data` volume — the Go service writes wav files to
  `/shared-data`, and the `tts-manager-http` sidecar
  (`voipbin-tts-manager-http`, `python3 -m http.server 80`) serves them,
  mirroring the GKE pod's http-server container. `POD_IP` is set to the
  sidecar's Compose service name so the media URL host resolves via Docker DNS
  on the `production` network (Asterisk fetches the wav from that URL).
  All three pieces (volume, sidecar, `POD_IP`) were dropped in the
  original Komodo migration, so batch TTS was silent on <prod-host> even
  with working credentials (same NOJIRA-Fix-tts-manager-gcp-credentials
  incident).
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

| Flag / Env Var | Description | Default |
|----------------|-------------|---------|
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server URL | `amqp://guest:guest@localhost:5672` |
| (no flag) / `AWS_ACCESS_KEY` | AWS access key id of Amazon Polly. Environment only, never a flag; read at the time of use. Not set, empty or an unresolved `[[...]]` placeholder means not configured; see [Provider keys](#provider-keys) | optional |
| (no flag) / `AWS_SECRET_KEY` | AWS secret access key of Amazon Polly. Same rules; the two AWS variables are used only together | optional |
| (no flag) / `ELEVENLABS_API_KEY` | ElevenLabs API key of the streaming handshake. Same rules; the same variable and value as in `bin-pipecat-manager` | optional |
| (no flag) / `TTS_CONFIG_ENCRYPTION_KEYS` | Master key list of the stored settings of the provider keys (not read by a request): comma-separated `<version>:<base64 32 bytes>`, the highest version encrypts. Environment only, never a flag. A malformed or empty value does not stop the start: the service logs the problem (never the value), refuses to store a secret, and reports the stored keys as undecryptable | optional |
| `gcp_tts_endpoint` / `GCP_TTS_ENDPOINT` | GCP TTS regional endpoint | `eu-texttospeech.googleapis.com:443` |
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `redis_address` / `REDIS_ADDRESS` | Redis server address | required |
| `redis_password` / `REDIS_PASSWORD` | Redis authentication | optional |
| `redis_db` / `REDIS_DB` | Redis DB index | optional |
| `POD_IP` | Media URL host for batch TTS. On GKE: pod IP (Downward API), rewritten to the pod-DNS form. On Docker Compose: the media http sidecar's Compose service name (`tts-manager-http`), used verbatim | required |
| `HOSTNAME` | Pod hostname (Kubernetes) — used as `HostID` for per-pod queue | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

GCP authentication uses Application Default Credentials via `GOOGLE_APPLICATION_CREDENTIALS`, pointing at `/run/secrets/google_service_account.json` — materialized by the Komodo compose `secrets:` block from the `BIN_MANAGER__GOOGLE_APPLICATION_CREDENTIALS_JSON` Komodo Variable (see the Deployment section above). If the credential file is missing or invalid, the daemon fails fast at startup instead of serving requests.

## Provider keys

The AWS Polly keys and the ElevenLabs key are read from environment variables, and only from there: `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` and `ELEVENLABS_API_KEY`. The stored settings (`aws.access_key`, `aws.secret_key`, `elevenlabs.api_key`, the `tts-control config` commands and the admin API) remain, but their values are not read by a request.

| Variable | Used by |
|----------|---------|
| `AWS_ACCESS_KEY` | Amazon Polly (the batch fallback and the AWS streaming provider), together with the secret key |
| `AWS_SECRET_KEY` | same |
| `ELEVENLABS_API_KEY` | The ElevenLabs streaming handshake |

A variable that is not set, is empty or still holds an unresolved deployment placeholder (a value that starts with `[[`) means that the key is not configured: an empty key and no error, and the placeholder is never sent to a provider. At start the service logs one line per variable (name and state only, never a value): an info line for a variable that is not set (every provider key is optional), and a warning with the words "unresolved placeholder" for a placeholder. Nothing is logged per call.

The two AWS variables are one credential. While exactly one of the two is set, AWS requests fail with an error (`ErrAWSPairIncomplete` wrapped by `ErrKeyUnavailable`; the batch request falls back to the other provider, a streaming message fails) and the start logs one warning. Set both or neither.

The variables are read for each request that needs them (nothing is cached), but the environment of a container changes only with a new deployment, so a rotation takes effect when the replicas restart. A batch request for the AWS provider that fails on AWS is served by GCP with its default voice (watch `tts_manager_speech_fallback_total` after a key change); a default request is served by GCP first and reaches AWS only as its fallback, so with a half AWS pair that fallback fails too. `tts-control create` reads the variables the same way.

### Stored settings

```bash
# the stored settings with their source (never a value); the service does not read them
./tts-control config list
# store a value (not used by the service): the value is read from stdin only (one line; at most 512 bytes) and is never printed
printf '%s\n' "$NEW_ACCESS_KEY" | ./tts-control config set --key aws.access_key --actor-id <uuid>
./tts-control config clear --key aws.access_key --actor-id <uuid>
```

`config set` and `config clear` print one line on stderr that names the variable that is read and says that the stored value is not used. The same keys can also be stored, deleted and read through the admin API of api-manager (`/service_configs/tts`, project super admin only), which reaches this service as `/v1/configs` requests on the shared queue. A change made that way is recorded in the history with the source `api` (the command line records `cli`). The request body of a secret is never logged. A stored value that is left behind is not rotated with the variables; an older image that reads the stored settings would use it.

The master key list is `TTS_CONFIG_ENCRYPTION_KEYS` (environment only, `<version>:<base64 of 32 bytes>[,...]`). Keep it in the secret store of the deployment, never in a flag or in the repository. The deployment definition must pass it to the container before `config set` can store a value; without it the commands list and clear but refuse to set, and a stored secret is reported as `db_undecryptable` by `config list`. The `config` commands need the database and the master key list only.

Shared variables, one value for one name:

- The AWS pair is also read by the transcription service (`AWS_ACCESS_KEY`, `AWS_SECRET_KEY`), so a rotation changes both services and the values must be equal in both. This is checked in the deployment variables, not in the code.
- `ELEVENLABS_API_KEY` is also read by the runner of `bin-pipecat-manager` from its own container. Rotate the deployment variable and redeploy both services.

Rotation:
1. Create the new key at the provider. Keep the old key valid.
2. Change the deployment variable (both AWS variables together) and redeploy this service and the other services that share the name. During a rolling deploy two replicas hold different values until the deploy ends.
3. Check the start log of each replica (the variables set, no placeholder warning) and place one test request.
4. Revoke the old key at the provider after the check. Keep it valid until the rollback window has ended: a rollback is the previous image together with the previous deployment definition. Stored values (`config set`) do not change what is used.

Recovery: a placeholder or a half AWS pair shows up in the start log; fix the deployment variable and redeploy. `tts-control config clear --key <key> --actor-id <uuid>` removes a stored value, which no request reads.

## Prometheus Metrics

Metrics exposed at `PROMETHEUS_LISTEN_ADDRESS` (default `:2112/metrics`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `tts_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `tts_manager_bucket_upload_process_time` | Histogram | — | Audio file upload duration |
| `tts_manager_bucket_url_process_time` | Histogram | — | Audio file URL generation duration |
| `tts_manager_gcp_synthesize_duration_seconds` | Histogram | — | GCP TTS synthesis duration |
| `tts_manager_hash_process_time` | Histogram | — | Text hash computation duration |
| `tts_manager_speech_create_duration_seconds` | Histogram | — | Total speech creation duration |
| `tts_manager_speech_fallback_total` | Counter | — | Number of fallback-to-AWS-Polly events |
| `tts_manager_speech_language_total` | Counter | `language` | Speech requests per language |
| `tts_manager_speech_request_total` | Counter | — | Total speech synthesis requests |
| `tts_manager_streaming_active` | Gauge | — | Currently active streaming sessions |
| `tts_manager_streaming_created_total` | Counter | — | Total streaming sessions created |
| `tts_manager_streaming_duration_seconds` | Histogram | — | Streaming session lifetime |
| `tts_manager_streaming_ended_total` | Counter | — | Total streaming sessions ended |
| `tts_manager_streaming_error_total` | Counter | — | Streaming errors (WebSocket disconnect, etc.) |
| `tts_manager_streaming_language_total` | Counter | `language` | Streaming sessions per language |
| `tts_manager_streaming_message_total` | Counter | — | Total streaming messages sent |
