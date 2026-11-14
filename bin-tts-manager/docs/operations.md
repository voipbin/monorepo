# bin-tts-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Service exits at startup ("Could not create the tts handler") | GCP ADC credentials not available — the daemon fails fast instead of nil-panicking on the first `/v1/speeches` request | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid credential file (`/run/secrets/google_service_account.json`, materialized by the Komodo compose `secrets:` block); verify the `BIN_MANAGER__GOOGLE_APPLICATION_CREDENTIALS_JSON` Komodo Variable is synced |
| Batch TTS returns no audio URL | GCP request failing at runtime (quota, endpoint), or `/shared-data` not writable (volume missing from the compose stack) | Check `speech_fallback_total` metric (AWS Polly is the fallback provider); verify the `shared-data` volume is mounted in `komodo/docker-compose.yml` |
| Call connects but TTS audio is silent while `/v1/speeches` succeeds | Asterisk cannot fetch the media URL — `voipbin-tts-manager-http` sidecar down, or `POD_IP` missing/wrong so the URL host does not resolve | Check the `tts-manager-http` container is running on the `production` network; verify `POD_IP=tts-manager-http` in the compose stack |
| AWS Polly fallback failing | Missing AWS keys, only one of the two AWS keys stored, or the stored keys can not be read | Run `tts-control config list` (see [Provider keys](#provider-keys)); `source` of both AWS keys must be `db`; check AWS Polly quotas |
| Streaming session RPC timeout | RPC routed to wrong pod (wrong per-pod queue) | Verify `host_id` matches `HOSTNAME` of target pod; check per-pod queue binding |
| AudioSocket connection refused | Go service port 8080 not listening | Check pod readiness; verify no port conflict with Python sidecar |
| Audio file not served by sidecar | `/shared-data` volume not mounted or empty | Check pod volume mount; verify Go service wrote the file before sidecar serves it |
| ElevenLabs WebSocket disconnect | Rate limit or API key invalid | Check `streaming_error_total` metric; verify the key with `tts-control config list` (`elevenlabs.api_key`, stored value only) |
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
| `aws_access_key` / `AWS_ACCESS_KEY` | Removed: the service does not read it any more. Store `aws.access_key` with `tts-control config set`; see [Provider keys](#provider-keys) | not read |
| `aws_secret_key` / `AWS_SECRET_KEY` | Removed: the service does not read it any more. Store `aws.secret_key` together with the access key | not read |
| `elevenlabs_api_key` / `ELEVENLABS_API_KEY` | Removed: this service does not read it any more (bin-pipecat-manager still reads it from its own environment). Store `elevenlabs.api_key` with `tts-control config set` | not read |
| (no flag) / `TTS_CONFIG_ENCRYPTION_KEYS` | Master key list of the stored provider keys: comma-separated `<version>:<base64 32 bytes>`, the highest version encrypts. Environment only, never a flag. A malformed or empty value does not stop the start: the service logs the problem (never the value), refuses to store a secret, and reports the stored keys as undecryptable | optional |
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

The AWS Polly keys and the ElevenLabs key are stored settings, read from the database only. A key without a stored row is not configured (an empty key); the environment variables `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` and `ELEVENLABS_API_KEY` are not read by this service any more (VOIP-1721). Enter the keys with `tts-control config set` or the admin API; `config clear` removes a key. The start logs an info line for each key that is not configured (key name only), because every provider key is optional.

| Stored key | Used by |
|------------|---------|
| `aws.access_key` | Amazon Polly (the batch fallback and the AWS streaming provider), together with the secret key |
| `aws.secret_key` | same |
| `elevenlabs.api_key` | The ElevenLabs streaming handshake |

The two AWS keys are one credential. While exactly one of the two AWS keys is stored, AWS requests fail (the batch request falls back to the other provider, a streaming message fails) and the start logs a warning. Store both keys, one after the other. This is a check of the sources, not a transaction: when both are stored and only one is changed, the pair mixes an old and a new value and AWS rejects the signature until the second key is stored.

The keys are read for each request that needs them (nothing is cached), so a change is used by the next request of every replica, with no restart. The price is one or two small database reads per AWS batch attempt (a cache miss), per AWS streaming message and per ElevenLabs streaming message, bounded to 5 seconds for the whole pair. While the database can not be read these requests fail; GCP requests do not read the database. A batch request for the AWS provider that fails on AWS is served by GCP with its default voice (watch `tts_manager_speech_fallback_total` after a key change); a default request is served by GCP first and reaches AWS only as its fallback, so with a mismatched AWS pair that fallback fails too. `tts-control create` reads the keys the same way, so it needs the database.

```bash
# the value is read from stdin only (one line; a key is at most 512 bytes) and is never printed
printf '%s\n' "$NEW_ACCESS_KEY" | ./tts-control config set --key aws.access_key --actor-id <uuid>
printf '%s\n' "$NEW_SECRET_KEY" | ./tts-control config set --key aws.secret_key --actor-id <uuid>
./tts-control config list
./tts-control config clear --key aws.access_key --actor-id <uuid>
```

The same keys can also be stored, deleted and read through the admin API of api-manager (`/service_configs/tts`, project super admin only), which reaches this service as `/v1/configs` requests on the shared queue. A change made that way is recorded in the history with the source `api` (the command line records `cli`). The request body of a secret is never logged.

The master key list is `TTS_CONFIG_ENCRYPTION_KEYS` (environment only, `<version>:<base64 of 32 bytes>[,...]`). Keep it in the secret store of the deployment, never in a flag or in the repository. The deployment definition must pass it to the container before `config set` can store a value; without it the commands list and clear but refuse to set, and a stored secret is reported as `db_undecryptable` (an error for its callers, never replaced by an empty key). The `config` commands need the database and the master key list only.

Other services read the same credentials from their own configuration and have separate stored settings:

- The AWS pair is also read by the transcription service. The two services keep separate stored settings, so the same AWS credential must be entered in both when it is rotated (in the same maintenance step). Check each service with its `config list`: `source` of both AWS keys. A mismatch shows as one service authenticating and the other failing at AWS, or as the warning at the start of this service that the two AWS sources differ.
- The ElevenLabs key is also read by the pipecat runner from the deployment variable `ELEVENLABS_API_KEY` as an environment value (it has no database); this service does not read that variable. Storing a new key here does not change the runner: replace the stored key and rotate the variable and redeploy the pipecat service too, and do not remove the variable while that service uses it.

Rotation:
1. Create the new key at the provider. Keep the old key valid.
2. Make sure every replica runs the image of this feature. Do not enter a value while a rolling deploy is half done, with the command line or the admin API: an older replica ignores the stored value (and does not serve the admin API), so the replicas would use different keys.
3. Store the new value (both AWS keys, one after the other), check with `config list`, and place one test request.
4. Revoke the old key at the provider. An image from before VOIP-1721 reads `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` and `ELEVENLABS_API_KEY` from the environment again when a key has no stored row; a credential rotated in the database only is not rotated for such an image. Keep the old key valid until the rollback window has ended. For such an image, the deployment does not inject the variables any more (VOIP-1730), so a rollback to such an image first restores those lines in the deployment definitions (revert of that change); this change does not delete the variables or secret keys of the deployment platform, so a restored line supplies the value again.

Recovery: `tts-control config clear --key <key> --actor-id <uuid>` removes the stored value; the key is then not configured until it is stored again.

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
