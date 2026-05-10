# voip-rtpengine-proxy — Dependencies

## Monorepo dependencies

| Module | Local path | Used for |
|--------|-----------|----------|
| `monorepo/bin-common-handler` | `../bin-common-handler` | `sockhandler` (RabbitMQ RPC), `sock` models |

This service has the smallest monorepo dependency footprint of any proxy — only `bin-common-handler` is required.

## External dependencies

| Package | Purpose |
|---------|---------|
| `github.com/sirupsen/logrus` | Structured logging |
| `github.com/joonix/log` | Fluentd-compatible JSON log formatter |
| `github.com/spf13/pflag` | CLI flag parsing |
| `github.com/spf13/viper` | Configuration binding (flags + env vars) |
| `github.com/prometheus/client_golang` | Prometheus metrics HTTP server |
| `github.com/go-redis/redis/v8` | Redis client for proxy address registration |
| `github.com/zeebo/bencode` | Bencode encode/decode for RTPEngine NG protocol |
| `cloud.google.com/go/storage` | GCS client for pcap uploads |
| `go.uber.org/mock` / `github.com/golang/mock` | Mock generation (test only) |

## Infrastructure dependencies

| Dependency | Default address | Purpose |
|-----------|----------------|---------|
| RabbitMQ | `amqp://guest:guest@localhost:5672` | Message bus for RPC request/response |
| Redis | `localhost:6379` (db 1) | Proxy address registration; upstream routing discovery |
| RTPEngine daemon | `127.0.0.1:22222` (UDP) | Co-located media proxy; receives NG protocol commands |
| GCS bucket | (optional) | pcap recording upload; disabled if `GCP_BUCKET_NAME_MEDIA` is unset |
