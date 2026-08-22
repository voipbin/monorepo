# bin-tts-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Service exits at startup ("Could not create the tts handler") | GCP ADC credentials not available — the daemon fails fast instead of nil-panicking on the first `/v1/speeches` request | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid credential file (`/run/secrets/google_service_account.json`, materialized by the Komodo compose `secrets:` block); verify the `BIN_MANAGER__GOOGLE_APPLICATION_CREDENTIALS_JSON` Komodo Variable is synced |
| Batch TTS returns no audio URL | GCP request failing at runtime (quota, endpoint), or `/shared-data` not writable (volume missing from the compose stack) | Check `speech_fallback_total` metric (AWS Polly is the fallback provider); verify the `shared-data` volume is mounted in `komodo/docker-compose.yml` |
| Call connects but TTS audio is silent while `/v1/speeches` succeeds | Asterisk cannot fetch the media URL — `voipbin-tts-manager-http` sidecar down, or `POD_IP` missing/wrong so the URL host does not resolve | Check the `tts-manager-http` container is running on the `production` network; verify `POD_IP=voipbin-tts-manager-http` in the compose stack |
| AWS Polly fallback failing | Missing `aws_access_key` / `aws_secret_key` | Verify credentials in environment; check AWS Polly quotas |
| Streaming session RPC timeout | RPC routed to wrong pod (wrong per-pod queue) | Verify `host_id` matches `HOSTNAME` of target pod; check per-pod queue binding |
| AudioSocket connection refused | Go service port 8080 not listening | Check pod readiness; verify no port conflict with Python sidecar |
| Audio file not served by sidecar | `/shared-data` volume not mounted or empty | Check pod volume mount; verify Go service wrote the file before sidecar serves it |
| ElevenLabs WebSocket disconnect | Rate limit or API key invalid | Check `streaming_error_total` metric; verify `ELEVENLABS_API_KEY` |
| Keep-alive timeout | Network issue between pod and ElevenLabs | Check `streaming_error_total`; session is cleaned up automatically |
| `POD_IP` not set | GKE: missing Downward API configuration. Compose: env dropped from the stack | GKE: verify `k8s/deployment.yml` injects `status.podIP` as `POD_IP`. Compose: verify `POD_IP=voipbin-tts-manager-http` in `komodo/docker-compose.yml` |

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
  bm-nyc-01 (see NOJIRA-Fix-tts-manager-gcp-credentials).
- **Batch-TTS media delivery:** the stack runs two containers sharing a
  named `shared-data` volume — the Go service writes wav files to
  `/shared-data`, and the `tts-manager-http` sidecar
  (`voipbin-tts-manager-http`, `python3 -m http.server 80`) serves them,
  mirroring the GKE pod's http-server container. `POD_IP` is set to the
  sidecar's container name so the media URL host resolves via Docker DNS
  on the `production` network (Asterisk fetches the wav from that URL).
  All three pieces (volume, sidecar, `POD_IP`) were dropped in the
  original Komodo migration, so batch TTS was silent on bm-nyc-01 even
  with working credentials (same NOJIRA-Fix-tts-manager-gcp-credentials
  incident).
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

| Flag / Env Var | Description | Default |
|----------------|-------------|---------|
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server URL | `amqp://guest:guest@localhost:5672` |
| `aws_access_key` / `AWS_ACCESS_KEY` | AWS Polly access key | optional |
| `aws_secret_key` / `AWS_SECRET_KEY` | AWS Polly secret key | optional |
| `elevenlabs_api_key` / `ELEVENLABS_API_KEY` | ElevenLabs API key (streaming) | required for streaming |
| `gcp_tts_endpoint` / `GCP_TTS_ENDPOINT` | GCP TTS regional endpoint | `eu-texttospeech.googleapis.com:443` |
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `redis_address` / `REDIS_ADDRESS` | Redis server address | required |
| `redis_password` / `REDIS_PASSWORD` | Redis authentication | optional |
| `redis_db` / `REDIS_DB` | Redis DB index | optional |
| `POD_IP` | Media URL host for batch TTS. On GKE: pod IP (Downward API), rewritten to the pod-DNS form. On Docker Compose: the media http sidecar's container name (`voipbin-tts-manager-http`), used verbatim | required |
| `HOSTNAME` | Pod hostname (Kubernetes) — used as `HostID` for per-pod queue | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

GCP authentication uses Application Default Credentials via `GOOGLE_APPLICATION_CREDENTIALS`, pointing at `/run/secrets/google_service_account.json` — materialized by the Komodo compose `secrets:` block from the `BIN_MANAGER__GOOGLE_APPLICATION_CREDENTIALS_JSON` Komodo Variable (see the Deployment section above). If the credential file is missing or invalid, the daemon fails fast at startup instead of serving requests.

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
