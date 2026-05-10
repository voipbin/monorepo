# voip-asterisk-proxy — Dependencies

## Monorepo service dependencies

| Module | Purpose |
|--------|---------|
| `monorepo/bin-agent-manager` | Agent model types used in call processing |
| `monorepo/bin-billing-manager` | Billing event types |
| `monorepo/bin-call-manager` | Call model types; primary consumer of ARI/AMI events |
| `monorepo/bin-common-handler` | Shared infrastructure: `sockhandler` (RabbitMQ), `notifyhandler` (event publish), `requesthandler` (RPC), common models |
| `monorepo/bin-talk-manager` | Talk session model types |
| `monorepo/bin-campaign-manager` | Campaign model types |
| `monorepo/bin-ai-manager` | AI integration model types |
| `monorepo/bin-conference-manager` | Conference model types |
| `monorepo/bin-conversation-manager` | Conversation model types |
| `monorepo/bin-email-manager` | Email model types |
| `monorepo/bin-flow-manager` | Flow model types |
| `monorepo/bin-hook-manager` | Webhook event types |
| `monorepo/bin-message-manager` | Message model types |
| `monorepo/bin-number-manager` | Number/DID model types |
| `monorepo/bin-outdial-manager` | Outbound dial model types |
| `monorepo/bin-pipecat-manager` | Pipecat AI session types |
| `monorepo/bin-queue-manager` | Queue model types |
| `monorepo/bin-registrar-manager` | SIP registrar model types |
| `monorepo/bin-route-manager` | Route model types |
| `monorepo/bin-storage-manager` | Storage model types |
| `monorepo/bin-tag-manager` | Tag model types |
| `monorepo/bin-transcribe-manager` | Transcription model types |
| `monorepo/bin-transfer-manager` | Transfer model types |
| `monorepo/bin-tts-manager` | TTS model types |
| `monorepo/bin-webhook-manager` | Webhook model types |
| `monorepo/bin-contact-manager` | Contact model types |
| `monorepo/bin-customer-manager` | Customer model types |
| `monorepo/bin-rag-manager` | RAG model types |
| `monorepo/bin-timeline-manager` | Timeline model types |

## External infrastructure dependencies

| Dependency | Purpose | Required |
|------------|---------|----------|
| **Asterisk PBX** | ARI (WebSocket on port 8088) and AMI (TCP on port 5038) | Yes — service has no purpose without Asterisk |
| **RabbitMQ** | Event publishing and RPC request consumption | Yes |
| **Redis** | Store proxy's internal IP address (looked up by upstream services) | Soft — failure degrades routing but does not crash the proxy |
| **Google Cloud Storage** | Store call recordings uploaded via `/proxy/recording_file_move` | Only if recording upload is used |
| **Kubernetes API** | Patch pod annotations with `asterisk-id` | Only if `KUBERNETES_DISABLED=false` (default) |

## Key third-party libraries

| Library | Purpose |
|---------|---------|
| `github.com/gorilla/websocket` | ARI WebSocket connection |
| `github.com/ivahaev/amigo` | AMI TCP client |
| `cloud.google.com/go/storage` | GCS recording upload |
| `github.com/go-redis/redis/v8` | Redis client |
| `github.com/spf13/viper` + `pflag` | Configuration management |
| `github.com/sirupsen/logrus` | Structured logging |
| `github.com/prometheus/client_golang` | Prometheus metrics |
