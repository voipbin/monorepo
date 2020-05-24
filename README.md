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
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_arievent string
        rabbitmq asterisk ari event queue name. (default "asterisk_ari_event")
  -rabbit_queue_flow string
        rabbitmq queue name for flow request (default "bin-manager.flow-manager.request")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.call-manager.request")
  -rabbit_queue_notify string
        rabbitmq queue name for event notify (default "bin-manager.call-manager.event")
  -worker_count int
        counts of workers (default 3)
```

## Example
```
$ ./call-manager -worker_count 5 -rabbit_addr "amqp://guest:guest@rabbitmq.voipbin.net:5672" -rabbit_queue_arievent asterisk_ari_event -prom_endpoint "/metrics" -prom_listen_addr ":2112" -dbDSN "call-manager:47f94686-8184-11ea-bfe8-e791e06ef5ef@tcp(10.126.80.5:3306)/bin_manager"
```

# RabbitMQ queues
## Request Listen Queue
bin-manager.call-manager.request

## Event Notify Queue
bin-manager.call-manager.event

# Note
