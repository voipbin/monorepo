# transcribe-manager

Speech-to-text service.

# Queues
* bin-manager.transcribe-manager.request : Listening queue for all transcribe-managers
* bin-manager.transcribe-manager-<uuid>.request : Listening queue for specific transcribe-manager.
* bin-manager.transcribe-manager.event : Event sending queue

# Example
```
./transcribe-manager \
        -gcp_bucket_name "voipbin-voip-media-bucket-europe-west4" \
        -gcp_credential "/home/pchero/service_accounts/google_service_account_voipbin_production.json" \
        -gcp_project_id "voipbin-production" \
        -prom_endpoint "/metrics" \
        -prom_listen_addr ":2112" \
        -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" \
        -rabbit_exchange_delay "bin-manager.delay" \
        -rabbit_queue_listen "bin-manager.transcribe-manager.request" \
        -rabbit_queue_notify "bin-manager.transcribe-manager.event" \
        -rabbit_queue_call "bin-manager.call-manager.request" \
        -rabbit_queue_webhook "bin-manager.webhook-manager.request" \
        -rabbit_queue_storage "bin-manager.storage-manager.request" \
        -redis_addr 10.164.15.220:6379 \
        -redis_db 1
```

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-transcribe-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-transcribe-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
