# call-manager
Handling the call.

# Usage
```
Usage of ./call-manager:
  -dbDSN string
        database dsn for call-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_queue_arievent string
        rabbitmq asterisk ari event queue name. (default "asterisk_ari_event")
  -rabbit_queue_arirequest string
        rabbitmq asterisk ari request queue prefix. (default "asterisk_ari_request")
```

## Example
```
$ ./call-manager -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" -rabbit_queue_arievent asterisk_ari_event -prom_endpoint "/metrics" -prom_listen_addr ":2112" -dbDSN "call-manager:47f94686-8184-11ea-bfe8-e791e06ef5ef@tcp(10.126.80.5:3306)/call_manager"
```

# Note
