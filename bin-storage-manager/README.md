# storage-manager

# Usage
```
Usage of ./storage-manager:
  -gcp_bucket_name string
        the gcp bucket name to use (default "bucket")
  -gcp_credential string
        the GCP credential file path (default "./credential.json")
  -gcp_project_id string
        the gcp project id (default "project")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_call string
        rabbitmq queue name for call request (default "bin-manager.call-manager.request")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.storage-manager.request")
  -rabbit_queue_notify string
        rabbitmq queue name for event notify (default "bin-manager.storage-manager.event")
```

# Example
```
./storage-manager \
	-gcp_bucket_name "voipbin-voip-media-bucket-europe-west4" \
	-gcp_credential "/home/pchero/service_accounts/google_service_account_voipbin_production.json" \
	-gcp_project_id "voipbin-production" \
	-prom_endpoint "/metrics" \
	-prom_listen_addr ":2112" \
	-rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" \
	-rabbit_exchange_delay "bin-manager.delay" \
	-rabbit_queue_listen "bin-manager.storage-manager.request" \
	-rabbit_queue_notify "bin-manager.storage-manager.event"
```

<!-- Updated dependencies: 2026-02-20 -->
