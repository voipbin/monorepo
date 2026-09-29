# asterisk-proxy

Bidirectional proxy between Asterisk PBX (ARI/AMI) and RabbitMQ. Connects to Asterisk's REST Interface (ARI) via WebSocket and Asterisk Manager Interface (AMI) via TCP, forwarding events to RabbitMQ and handling RPC requests to control Asterisk.

# Configuration

All flags can also be set via environment variables.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--ari_address` | `ARI_ADDRESS` | `localhost:8088` | Address of the ARI server |
| `--ari_account` | `ARI_ACCOUNT` | `asterisk:asterisk` | ARI account (id:password) |
| `--ari_subscribe_all` | `ARI_SUBSCRIBE_ALL` | `true` | Subscribe to all ARI events |
| `--ari_application` | `ARI_APPLICATION` | `voipbin` | ARI application name |
| `--ami_host` | `AMI_HOST` | `127.0.0.1` | AMI server host |
| `--ami_port` | `AMI_PORT` | `5038` | AMI server port |
| `--ami_username` | `AMI_USERNAME` | `asterisk` | AMI username |
| `--ami_password` | `AMI_PASSWORD` | `asterisk` | AMI password |
| `--ami_event_filter` | `AMI_EVENT_FILTER` | `` | AMI event filter |
| `--interface_name` | `INTERFACE_NAME` | `eth0` | Network interface for identity (MAC address) |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server address |
| `--rabbitmq_queue_listen` | `RABBITMQ_QUEUE_LISTEN` | `asterisk.call.request` | RabbitMQ listen queue name |
| `--redis_address` | `REDIS_ADDRESS` | `localhost:6379` | Redis server address |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis database index |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Prometheus metrics endpoint path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Prometheus listen address |
| `--recording_bucket_name` | `RECORDING_BUCKET_NAME` | `` | GCS bucket name for recording uploads (required for uploads) |
| `--recording_asterisk_directory` | `RECORDING_ASTERISK_DIRECTORY` | `/var/spool/asterisk/recording` | Local Asterisk recording directory (shared with the Asterisk container) |
| `--recording_bucket_directory` | `RECORDING_BUCKET_DIRECTORY` | `/mnt/media/recording` | GCS object-name prefix, used verbatim as `<prefix>/<filename>` (not a filesystem mount). The default yields object names starting with `/`; VoIPBin deployments set `recording`, the prefix call-manager uses when it registers the file with storage-manager |
| `--google_application_credentials_json` | `GOOGLE_APPLICATION_CREDENTIALS_JSON` | `` | GCP service-account key JSON content (not a file path) for GCS uploads. Empty means Application Default Credentials |
| `--kubernetes_disabled` | `KUBERNETES_DISABLED` | `false` | Disable Kubernetes integration |

# Run

```bash
./asterisk-proxy \
  --ari_address localhost:8088 \
  --ari_account asterisk:asterisk \
  --ari_application voipbin \
  --ari_subscribe_all true \
  --ami_host 127.0.0.1 \
  --ami_port 5038 \
  --ami_username asterisk \
  --ami_password asterisk \
  --interface_name eth0 \
  --rabbitmq_address amqp://guest:guest@localhost:5672 \
  --rabbitmq_queue_listen asterisk.call.request \
  --redis_address localhost:6379 \
  --redis_database 1
```

Or use environment variables:
```bash
export ARI_ADDRESS=localhost:8088
export RABBITMQ_ADDRESS=amqp://guest:guest@localhost:5672
./asterisk-proxy
```

# Recording upload

The proxy uploads recordings itself with the Google Cloud Storage Go client when it receives
the `/proxy/recording_file_move` RPC. The bucket is not mounted (no gcsfuse); only the local
recording directory is shared with the Asterisk container.

For each filename, in order:
1. Open `<recording_asterisk_directory>/<filename>`.
2. Copy it into a GCS object writer for object `<recording_bucket_directory>/<filename>` in bucket
   `<recording_bucket_name>`. The object name is the two values joined with `/`, without normalization.
3. Close the object writer. GCS commits the upload here, and the result is checked.
4. Delete the local file, only after the upload is committed.

Failure behavior:
- Open or copy error: the upload is aborted (no partial object), the local file is kept, and the
  RPC returns an error.
- Upload rejected at commit (permissions, network, revoked key, missing bucket): the local file is
  kept and the RPC returns an error. The caller does not register the recording.
- Local delete error after a committed upload: the RPC returns success (the recording is stored
  and gets registered) and the proxy logs an error with the source and destination paths. When the
  recording is registered, storage-manager moves the object from `<recording_bucket_directory>/<filename>`
  to its own `bin/<file id>` path, so it is no longer at the logged destination. Remove the local
  file manually once the recording's storage file record exists.
- Files are processed in request order and processing stops at the first error, so earlier files
  in the same request may already be uploaded and deleted.

Kept local files survive only as long as the recording volume does: a named Docker volume (as in
the Komodo deployment) survives container recreation but not volume deletion or stack
migration; a Kubernetes `emptyDir` is lost when the pod is deleted.

If the GCS client cannot be created at startup (for example, invalid credential JSON, or no
credential JSON and no Application Default Credentials), the proxy still starts and only recording
upload returns an error.

# RabbitMQ RPC

Event message
```
	Type     string `json:"type"`
	DataType string `json:"data_type"`
	Data     string `json:"data"`
```

ARI event
```json
{
  "type": "ari_event",
  "data_type": "application/json",
  "data": "{...}"
}
```

AMI event
```json
{
  "type": "ami_event",
  "data_type": "application/json",
  "data": "{...}"
}
```

RPC requests

ARI request
```json
{
  "uri": "/ari/channels?api_key=asterisk:asterisk&endpoint=pjsip/test@sippuas&app=test",
  "method": "POST",
  "data": "data",
  "data_type": "text/plain"
}
```

AMI request
```json
{
  "uri": "/ami",
  "method": "",
  "data": "{\"Action\": \"Ping\"}",
  "data_type": "text/plain"
}
```

RPC response
```json
{
  "status_code": 200,
  "data_type": "application/json",
  "data": "{...}"
}
```

<!-- Updated dependencies: 2026-02-20 -->
