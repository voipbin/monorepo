# voip-kamailio-proxy — Dependencies

## Monorepo dependencies

The `go.mod` declares `replace` directives for the following monorepo modules:

| Module | Local path | Used for |
|--------|-----------|----------|
| `monorepo/bin-common-handler` | `../bin-common-handler` | `sockhandler` (RabbitMQ RPC), `sock` models |
| `monorepo/bin-agent-manager` | `../bin-agent-manager` | Transitive (not directly imported by proxy code) |
| `monorepo/bin-call-manager` | `../bin-call-manager` | Transitive |
| `monorepo/bin-talk-manager` | `../bin-talk-manager` | Transitive |
| `monorepo/bin-billing-manager` | `../bin-billing-manager` | Transitive |
| `monorepo/bin-campaign-manager` | `../bin-campaign-manager` | Transitive |
| `monorepo/bin-ai-manager` | `../bin-ai-manager` | Transitive |
| `monorepo/bin-conference-manager` | `../bin-conference-manager` | Transitive |
| `monorepo/bin-conversation-manager` | `../bin-conversation-manager` | Transitive |
| `monorepo/bin-email-manager` | `../bin-email-manager` | Transitive |
| `monorepo/bin-flow-manager` | `../bin-flow-manager` | Transitive |
| `monorepo/bin-hook-manager` | `../bin-hook-manager` | Transitive |
| `monorepo/bin-message-manager` | `../bin-message-manager` | Transitive |
| `monorepo/bin-number-manager` | `../bin-number-manager` | Transitive |
| `monorepo/bin-outdial-manager` | `../bin-outdial-manager` | Transitive |
| `monorepo/bin-pipecat-manager` | `../bin-pipecat-manager` | Transitive |
| `monorepo/bin-queue-manager` | `../bin-queue-manager` | Transitive |
| `monorepo/bin-registrar-manager` | `../bin-registrar-manager` | Transitive |
| `monorepo/bin-route-manager` | `../bin-route-manager` | Transitive |
| `monorepo/bin-storage-manager` | `../bin-storage-manager` | Transitive |
| `monorepo/bin-tag-manager` | `../bin-tag-manager` | Transitive |
| `monorepo/bin-transcribe-manager` | `../bin-transcribe-manager` | Transitive |
| `monorepo/bin-transfer-manager` | `../bin-transfer-manager` | Transitive |
| `monorepo/bin-tts-manager` | `../bin-tts-manager` | Transitive |
| `monorepo/bin-webhook-manager` | `../bin-webhook-manager` | Transitive |
| `monorepo/bin-contact-manager` | `../bin-contact-manager` | Transitive |
| `monorepo/bin-customer-manager` | `../bin-customer-manager` | Transitive |
| `monorepo/bin-rag-manager` | `../bin-rag-manager` | Transitive |
| `monorepo/bin-timeline-manager` | `../bin-timeline-manager` | Transitive |
| `monorepo/bin-direct-manager` | `../bin-direct-manager` | Transitive |
| `monorepo/bin-sentinel-manager` | `../bin-sentinel-manager` | Transitive |

The large transitive replace list is inherited from `bin-common-handler`'s own module graph. The proxy itself only directly imports `bin-common-handler`.

## External dependencies

| Package | Purpose |
|---------|---------|
| `github.com/sirupsen/logrus` | Structured logging |
| `github.com/joonix/log` | Fluentd-compatible JSON log formatter |
| `github.com/spf13/pflag` | CLI flag parsing |
| `github.com/spf13/viper` | Configuration binding (flags + env vars) |
| `github.com/prometheus/client_golang` | Prometheus metrics HTTP server |

## Infrastructure dependencies

| Dependency | Purpose |
|-----------|---------|
| RabbitMQ | Message bus for RPC request/response |
| Kamailio daemon | Co-located SIP proxy (not communicated with by Go; shares the pod) |

The service does not depend on MySQL, Redis, or any other storage system. All state is transient.
