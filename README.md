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
        rabbitmq asterisk ari event queue name. (default "asterisk.all.event")
  -rabbit_queue_flow string
        rabbitmq queue name for flow request (default "bin-manager.flow-manager.request")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.call-manager.request")
  -rabbit_queue_notify string
        rabbitmq queue name for event notify (default "bin-manager.call-manager.event")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
```

## Example
```
$ ./call-manager \
-prom_endpoint /metrics \
-prom_listen_addr :2112 \
-dbDSN 'bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager' \
-rabbit_addr amqp://guest:guest@rabbitmq.voipbin.net:5672 \
-rabbit_queue_listen bin-manager.call-manager.request \
-rabbit_queue_notify bin-manager.call-manager.event \
-rabbit_exchange_delay bin-manager.delay \
-rabbit_queue_arievent asterisk.all.event \
-rabbit_queue_flow bin-manager.flow-manager.request \
-redis_addr 10.164.15.220:6379 \
-redis_db 1
```

# RabbitMQ queues
## Request Listen Queue
bin-manager.call-manager.request

## Event Notify Queue
bin-manager.call-manager.event

# Note
