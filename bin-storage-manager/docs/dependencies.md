# bin-storage-manager Dependencies

## Upstream Services (consumed via RabbitMQ RPC)

| Service | Purpose |
|---------|---------|
| `bin-billing-manager` | Billing checks for storage operations |
| `bin-customer-manager` | Customer validation; source of `customer_deleted` events |

## Events Subscribed

| Queue | Event | Handler |
|-------|-------|---------|
| `bin-manager.customer-manager.event` | `customer_deleted` | Cascading delete of all accounts and files for the customer |

## Infrastructure Dependencies

| Dependency | Use |
|-----------|-----|
| RabbitMQ | RPC request queue (`bin-manager.storage-manager.request`) and event subscription |
| MySQL | Durable storage for file and account records |
| Redis | Cache for file and account lookups; invalidated on mutation |
| GCS bucket (`gcp_bucket_name_media`) | Persistent storage for recordings and uploaded files |
| GCS bucket (`gcp_bucket_name_tmp`) | Transient storage for on-demand zip archives |
| Google IAM Credentials API | Signed URL generation in GKE without a local service account key |

## Monorepo Module Dependencies

Key local imports:
- `monorepo/bin-common-handler` — sockhandler, requesthandler, notifyhandler, databasehandler
- `monorepo/bin-customer-manager` — customer event models used by SubscribeHandler

All other `bin-*` replace directives in `go.mod` allow cross-service type sharing without creating import cycles.

## Reverse Dependencies

Many services call `bin-storage-manager` to store or retrieve files:
- `bin-call-manager` — recording upload after call ends
- `bin-tts-manager` — TTS output file storage
- `bin-rag-manager` — retrieves source documents for embedding
- `bin-transcribe-manager` — retrieves audio files for transcription
