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
| `--recording_bucket_name` | `RECORDING_BUCKET_NAME` | `` | GCS bucket name for recordings |
| `--recording_asterisk_directory` | `RECORDING_ASTERISK_DIRECTORY` | `/var/spool/asterisk/recording` | Asterisk recording directory |
| `--recording_bucket_directory` | `RECORDING_BUCKET_DIRECTORY` | `/mnt/media/recording` | Bucket recording directory |
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
