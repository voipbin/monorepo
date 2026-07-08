# bin-transcribe-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Session stuck in `progressing` | `call_hangup` event not received or not processed | Check subscribe handler queue; manually stop via `POST /v1/transcribes/{id}/stop` |
| STT RPC routed to wrong pod | Stale `host_id` after pod restart (Calico POD_IP recycle) | See [per-pod-queues.md](../../docs/patterns/per-pod-queues.md) for known limitation; session must be recreated |
| No transcripts appearing | WebSocket to Asterisk not established | Check `streaming_handler` logs for dial errors; verify `MediaURI` from `ExternalMediaStart` |
| GCP auth failure | ADC not configured | Check `GOOGLE_APPLICATION_CREDENTIALS` points to a valid mounted service account key file |
| AWS auth failure | Missing credentials | Verify `AWS_ACCESS_KEY` / `AWS_SECRET_KEY` env vars |
| Session health-check failing | Pod hosting session is down | Session is lost; streaming cannot be resumed — client must recreate |
| `customer_deleted` cascade not running | Subscribe handler not consuming customer-manager events | Check queue binding: `bin-manager.customer-manager.event` |

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

GCP authentication uses Application Default Credentials. At least one STT provider (GCP or AWS) must be configured.

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
