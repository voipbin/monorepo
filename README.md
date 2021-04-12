# stt-manager

Speech-to-text service

# Example
```
./stt-manager \
        -gcp_bucket_name "voipbin-voip-media-bucket-europe-west4" \
        -gcp_credential "/home/pchero/service_accounts/google_service_account_voipbin_production.json" \
        -gcp_project_id "voipbin-production" \
        -prom_endpoint "/metrics" \
        -prom_listen_addr ":2112" \
        -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" \
        -rabbit_exchange_delay "bin-manager.delay" \
        -rabbit_queue_listen "bin-manager.stt-manager.request" \
        -rabbit_queue_notify "bin-manager.stt-manager.event" \
        -rabbit_queue_call "bin-manager.call-manager.request" \
        -rabbit_queue_webhook "bin-manager.webhook-manager.request" \
        -rabbit_queue_storage "bin-manager.storage-manager.request" \
        -redis_addr 10.164.15.220:6379 \
        -redis_db 1
```
