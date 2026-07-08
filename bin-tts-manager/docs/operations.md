# bin-tts-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Batch TTS returns no audio URL | GCP ADC credentials not available | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid mounted service account key file; check `speech_fallback_total` metric |
| AWS Polly fallback failing | Missing `aws_access_key` / `aws_secret_key` | Verify credentials in environment; check AWS Polly quotas |
| Streaming session RPC timeout | RPC routed to wrong pod (wrong per-pod queue) | Verify `host_id` matches `HOSTNAME` of target pod; check per-pod queue binding |
| AudioSocket connection refused | Go service port 8080 not listening | Check pod readiness; verify no port conflict with Python sidecar |
| Audio file not served by sidecar | `/shared-data` volume not mounted or empty | Check pod volume mount; verify Go service wrote the file before sidecar serves it |
| ElevenLabs WebSocket disconnect | Rate limit or API key invalid | Check `streaming_error_total` metric; verify `ELEVENLABS_API_KEY` |
| Keep-alive timeout | Network issue between pod and ElevenLabs | Check `streaming_error_total`; session is cleaned up automatically |
| `POD_IP` not set | Missing Kubernetes Downward API configuration | Verify `k8s/deployment.yml` injects `status.podIP` as `POD_IP` |

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
| `POD_IP` | Pod IP (Kubernetes Downward API) — AudioSocket advertise address | required |
| `HOSTNAME` | Pod hostname (Kubernetes) — used as `HostID` for per-pod queue | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

GCP authentication uses Application Default Credentials via `GOOGLE_APPLICATION_CREDENTIALS` (mounted service account key file from `Secret/voipbin` key `GOOGLE_APPLICATION_CREDENTIALS_JSON`).

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
