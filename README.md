# call-manager
Recevies ARI events

# Usage
```
Usage of ./call-manager:
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
$  ./call-manager -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" -rabbit_queue_arievent asterisk_ari_event -prom_endpoint "/metrics" -prom_listen_addr ":2112"
```

# Note

